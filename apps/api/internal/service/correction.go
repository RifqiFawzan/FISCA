package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// correction is a parsed instruction to change an existing transaction, for
// example "ubah kopi jadi 15rb" or "salah, seharusnya 15rb" (PRD §22.3).
type correction struct {
	match       bool
	target      string // description fragment to find; "" means newest
	amount      *int64
	description string // replacement description, "" keeps the current one
}

var (
	// "ubah|ganti|rubah|koreksi|edit <sesuatu> jadi|ke <nilai>"
	correctionChangeRe = regexp.MustCompile(
		`(?i)^(?:tolong\s+|coba\s+|mohon\s+)?(?:ubah|ganti|rubah|koreksi|edit)\s+(.+?)\s+(?:jadi|ke)\s+(.+)$`)
	// "salah, seharusnya <nilai>"
	correctionFixRe = regexp.MustCompile(
		`(?i)^(?:salah|maaf|koreksi)\s*,?\s*(?:seharusnya|harusnya|harus)\s+(.+)$`)
)

// correctionStopwords keep account, budget and navigation sentences out of the
// correction path: those have their own handlers.
var correctionStopwords = []string{
	"budget", "anggaran", "kategori", "akun", "profil", "pengaturan",
	"riwayat", "cari", "berapa", "laporan",
}

// detectCorrection reports whether a message asks to change an existing
// transaction and, if so, what should change.
func detectCorrection(text string) correction {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, stop := range correctionStopwords {
		if strings.Contains(lower, stop) {
			return correction{}
		}
	}

	var lhs, rhs string
	switch {
	case correctionChangeRe.MatchString(lower):
		match := correctionChangeRe.FindStringSubmatch(lower)
		lhs, rhs = match[1], match[2]
	case correctionFixRe.MatchString(lower):
		rhs = correctionFixRe.FindStringSubmatch(lower)[1]
	default:
		return correction{}
	}

	amount, description := parseCorrectionValue(rhs)
	target := stripFiller(lhs)
	if amount == nil && description == "" {
		return correction{}
	}
	return correction{match: true, target: target, amount: amount, description: description}
}

// parseCorrectionValue reads the right-hand side of a correction: a nominal
// ("15rb"), a description ("teh"), or both ("teh 15rb").
func parseCorrectionValue(raw string) (*int64, string) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	var amount *int64
	if parsed, ok := parseAmountFromText(lower); ok {
		amount = &parsed
		lower = strings.TrimSpace(anyAmountRe.ReplaceAllString(lower, " "))
	}
	return amount, cleanDescription(stripDateWords(lower))
}

// stripFiller reduces the left-hand side to the words that identify the row:
// "kopi tadi" and "beli makan kemarin" both become a description fragment.
func stripFiller(raw string) string {
	out := strings.ToLower(strings.TrimSpace(raw))
	out = anyAmountRe.ReplaceAllString(out, " ")
	out = stripDateWords(out)
	for _, filler := range []string{"tadi", "tersebut", "sebelumnya", "yang", "terakhir", "ini"} {
		out = strings.ReplaceAll(out, filler, " ")
	}
	return strings.Join(strings.Fields(out), " ")
}

// findCorrectionTarget picks the newest active transaction matching the
// fragment, or the newest transaction when no fragment was given.
func findCorrectionTarget(items []TransactionDTO, target string) *TransactionDTO {
	if len(items) == 0 {
		return nil
	}
	if target == "" {
		return &items[0]
	}
	for i := range items {
		if strings.Contains(strings.ToLower(items[i].Description), target) {
			return &items[i]
		}
	}
	return nil
}

// applyCorrection edits an existing transaction through the versioned update
// path, so the audit trail and the optimistic lock behave exactly as they do
// for a manual edit.
func (s *Service) applyCorrection(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	corr correction,
	wc WriteContext,
) (string, Card, error) {
	page, err := s.ListTransactions(ctx, q, userID, ListTransactionsInput{Limit: 25})
	if err != nil {
		return "", Card{}, err
	}

	target := findCorrectionTarget(page.Items, corr.target)
	if target == nil {
		body := "Transaksi yang cocok tidak ditemukan."
		if corr.target != "" {
			body = fmt.Sprintf("Tidak ada transaksi berdeskripsi %q. Sebutkan deskripsinya, misalnya: ubah kopi jadi 15rb.", corr.target)
		}
		return body, Card{
			Type:          "clarification",
			Title:         "Transaksi tidak ditemukan",
			Body:          body,
			MissingFields: []string{"target"},
			NeedsReply:    true,
		}, nil
	}

	in := UpdateTransactionInput{Version: target.Version}
	if corr.amount != nil {
		in.Amount = corr.amount
	}
	if corr.description != "" {
		in.Description = &corr.description
	}
	if in.Amount == nil && in.Description == nil {
		body := "Sebutkan nominal atau deskripsi barunya, misalnya: ubah kopi jadi 15rb."
		return body, Card{
			Type:          "clarification",
			Title:         "Koreksi belum jelas",
			Body:          body,
			MissingFields: []string{"amount"},
			NeedsReply:    true,
		}, nil
	}

	updated, err := s.UpdateTransaction(ctx, q, userID, target.ID, in, wc)
	if err != nil {
		return "", Card{}, err
	}

	reply := fmt.Sprintf("%s dikoreksi jadi Rp%s.", updated.Description, formatRupiahPlain(updated.Amount))
	return reply, Card{
		Type:        "transaction_result",
		Title:       "Transaksi dikoreksi",
		Body:        reply,
		Transaction: &updated,
	}, nil
}
