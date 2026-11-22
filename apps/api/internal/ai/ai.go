// Package ai is the only place that talks to a language model (plan.md §9).
//
// The model is never trusted to write money: it returns an intent, the service
// validates every field, and only deterministic code writes transaction rows.
// When OPENAI_API_KEY is absent the assistant uses local heuristics instead of
// returning an error, so the product still works offline.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// ErrUnavailable means no API key is configured.
var ErrUnavailable = errors.New("ai unavailable")

// Intent is the structured meaning of one user message.
type Intent struct {
	Kind         string  `json:"kind"` // create_transaction | spending_summary | largest_transactions | search | budget_status | proposal | unknown
	Type         string  `json:"type"`
	Amount       int64   `json:"amount"`
	Description  string  `json:"description"`
	CategoryName string  `json:"categoryName"`
	AccountName  string  `json:"accountName"`
	Period       string  `json:"period"`
	Query        string  `json:"query"`
	Rationale    string  `json:"rationale"`
	Confidence   float64 `json:"confidence"`
}

// Context carries the user's vocabulary so the model can only choose real
// categories and accounts.
type Context struct {
	Categories []string
	Accounts   []string
	Today      time.Time
}

// Client is a minimal OpenAI REST client (chat completions + transcriptions).
type Client struct {
	APIKey string
	// BaseURL serves chat completions. STTBaseURL, when set, serves only the
	// transcription endpoint so a local speech-to-text server can be used
	// alongside a hosted chat model.
	BaseURL    string
	STTBaseURL string
	Model      string
	STT        string
	Timeout    time.Duration
	HTTP       *http.Client
}

// New builds a Client. An empty key yields a client that reports Available()
// false and never makes network calls, unless sttBaseURL points at a local
// transcription server, which needs no key to run.
func New(apiKey, baseURL, sttBaseURL, model, stt string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Client{
		APIKey:     strings.TrimSpace(apiKey),
		BaseURL:    strings.TrimRight(baseURL, "/"),
		STTBaseURL: strings.TrimRight(sttBaseURL, "/"),
		Model:      model,
		STT:        stt,
		Timeout:    timeout,
		HTTP:       &http.Client{Timeout: timeout},
	}
}

// Available reports whether the client can call the API.
func (c *Client) Available() bool { return c != nil && c.APIKey != "" }

// STTAvailable reports whether transcription can be attempted. A configured
// local speech-to-text server needs no key, so it counts on its own.
func (c *Client) STTAvailable() bool {
	return c != nil && (c.STTBaseURL != "" || c.APIKey != "")
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ExtractIntent asks the model for a JSON intent.
func (c *Client) ExtractIntent(ctx context.Context, text string, cctx Context) (Intent, error) {
	if !c.Available() {
		return Intent{}, ErrUnavailable
	}

	categoryList := "-"
	accountList := "-"
	if len(cctx.Categories) > 0 {
		categoryList = strings.Join(cctx.Categories, ", ")
	}
	if len(cctx.Accounts) > 0 {
		accountList = strings.Join(cctx.Accounts, ", ")
	}

	system := `Kamu adalah parser intent untuk aplikasi keuangan pribadi Indonesia.
Balas HANYA JSON valid dengan skema:
{"kind":"create_transaction|spending_summary|largest_transactions|search|budget_status|proposal|unknown",
 "type":"income|expense|transfer|",
 "amount":0,
 "description":"",
 "categoryName":"",
 "accountName":"",
 "period":"this_month|last_month|7d|30d|90d|this_year|YYYY-MM|",
 "query":"",
 "rationale":"",
 "confidence":0.0}

Aturan:
- Uang selalu integer rupiah. "25rb" = 25000, "1,5jt" = 1500000.
- Jangan pernah mengarang nominal. Bila tidak ada nominal, amount 0.
- categoryName hanya boleh dari daftar kategori pengguna.
- accountName hanya boleh dari daftar akun pengguna.
- "catat", "beli", "bayar", "terima", "gaji" biasanya create_transaction.
- "berapa", "total", "ringkasan", "pengeluaran bulan ini" = spending_summary.
- "paling besar", "termahal" = largest_transactions.
- "cari", "riwayat", "transaksi ..." = search.
- "budget", "anggaran" = budget_status atau proposal.
- Hari ini: ` + cctx.Today.Format("2006-01-02") + `

Kategori pengguna: ` + categoryList + `
Akun pengguna: ` + accountList

	body := chatRequest{
		Model:       c.Model,
		Temperature: 0.1,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: text},
		},
		ResponseFormat: map[string]any{"type": "json_object"},
	}
	raw, err := c.postChat(ctx, body)
	if err != nil {
		return Intent{}, err
	}

	var intent Intent
	if err := json.Unmarshal(raw, &intent); err != nil {
		return Intent{}, fmt.Errorf("parse intent: %w", err)
	}
	if intent.Kind == "" {
		intent.Kind = "unknown"
	}
	return intent, nil
}

func (c *Client) postChat(ctx context.Context, body chatRequest) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var parsed chatResponse
	_ = json.Unmarshal(data, &parsed)
	if resp.StatusCode >= 300 {
		message := "ai provider error"
		if parsed.Error != nil && parsed.Error.Message != "" {
			message = parsed.Error.Message
		}
		return nil, fmt.Errorf("ai provider %d: %s", resp.StatusCode, message)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("ai returned no choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return []byte(strings.TrimSpace(content)), nil
}

// Transcribe sends audio to the transcription endpoint and returns the text.
//
// The endpoint is OpenAI's /audio/transcriptions contract, so any
// OpenAI-compatible server works. STTBaseURL redirects this call alone, which
// lets chat stay on a hosted model while audio is transcribed locally.
func (c *Client) Transcribe(ctx context.Context, filename string, audio io.Reader) (string, error) {
	base := c.BaseURL
	if c.STTBaseURL != "" {
		base = c.STTBaseURL
	} else if !c.Available() {
		return "", ErrUnavailable
	}
	if base == "" {
		return "", ErrUnavailable
	}
	if filename == "" {
		filename = "audio.webm"
	}

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, io.LimitReader(audio, 25<<20)); err != nil {
		return "", err
	}
	_ = writer.WriteField("model", c.STT)
	_ = writer.WriteField("language", "id")
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/audio/transcriptions", buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcription request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("transcription provider %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("parse transcription: %w", err)
	}
	return strings.TrimSpace(out.Text), nil
}
