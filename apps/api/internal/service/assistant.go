package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fisca-app/fisca/apps/api/internal/ai"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// AssistantRequest is the POST /assistant/messages body.
type AssistantRequest struct {
	ConversationID string `json:"conversationId"`
	Text           string `json:"text"`
	Source         string `json:"source"` // text | voice
}

// Card is the visual contract consumed by the assistant screen (plan.md §7.3).
type Card struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Body       string `json:"body,omitempty"`
	Amount     int64  `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
	NeedsReply bool   `json:"needsReply,omitempty"`

	Transaction *TransactionDTO     `json:"transaction,omitempty"`
	Summary     *SummaryDTO         `json:"summary,omitempty"`
	Period      *PeriodDTO          `json:"period,omitempty"`
	Budget      *BudgetDTO          `json:"budget,omitempty"`
	Proposal    *BudgetProposalDTO  `json:"proposal,omitempty"`
	Unusual     *UnusualSpendingDTO `json:"unusual,omitempty"`

	Categories []CategorySpendDTO `json:"categories,omitempty"`
	// Accounts menopang kartu saldo: daftar nama + saldo per akun hasil
	// hitungan backend (ListAccountBalances), bukan angka karangan model.
	Accounts []AccountDTO `json:"accounts,omitempty"`
	// Trend meneruskan monthlyTrend backend untuk chart mini di kartu
	// ringkasan (analisis interaktif; angka tetap hasil hitungan backend).
	Trend         []MonthlyTotalDTO `json:"trend,omitempty"`
	Items         []TransactionDTO  `json:"items,omitempty"`
	Warnings      []Warning         `json:"warnings,omitempty"`
	MissingFields []string          `json:"missingFields,omitempty"`

	DraftID   string          `json:"draftId,omitempty"`
	DraftData json.RawMessage `json:"draft,omitempty"`
}

// AssistantResponse is what the UI renders.
type AssistantResponse struct {
	ConversationID string   `json:"conversationId"`
	MessageID      string   `json:"messageId"`
	Reply          string   `json:"reply"`
	Card           Card     `json:"card"`
	ToolTrace      []string `json:"toolTrace"`
}

// ConversationDTO is one thread in the sidebar.
type ConversationDTO struct {
	ID            string  `json:"id"`
	Title         *string `json:"title"`
	CreatedAt     string  `json:"createdAt"`
	LastMessageAt string  `json:"lastMessageAt"`
}

// MessageDTO is one persisted message.
type MessageDTO struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Content   *string         `json:"content"`
	Card      json.RawMessage `json:"card,omitempty"`
	CreatedAt string          `json:"createdAt"`
}

// DraftDTO is an unresolved clarification, never part of any aggregate.
type DraftDTO struct {
	ID            string   `json:"id"`
	MissingFields []string `json:"missingFields"`
	Status        string   `json:"status"`
}

var anyAmountRe = regexp.MustCompile(`(?i)(?:rp\.?\s*)?([0-9][0-9.,]*)\s*(rb|ribu|k|jt|juta)?`)

// HandleMessage interprets a chat/voice message, performs the action, stores
// both messages, and returns a card.
func (s *Service) HandleMessage(ctx context.Context, q *generated.Queries, userID uuid.UUID, in AssistantRequest, wc WriteContext) (AssistantResponse, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return AssistantResponse{}, Invalid("text", "Pesan tidak boleh kosong")
	}
	if len([]rune(text)) > 2000 {
		return AssistantResponse{}, Invalid("text", "Pesan maksimal 2000 karakter")
	}
	if in.Source == "" {
		in.Source = "text"
	}
	if in.Source != "text" && in.Source != "voice" {
		return AssistantResponse{}, Invalid("source", "Source harus text atau voice")
	}

	conversation, err := s.ensureConversation(ctx, q, userID, in.ConversationID, text)
	if err != nil {
		return AssistantResponse{}, err
	}

	if _, err := q.CreateAssistantMessage(ctx, generated.CreateAssistantMessageParams{
		ConversationID: conversation.ID,
		UserID:         pgUUIDValue(userID),
		Role:           "user",
		Content:        &text,
	}); err != nil {
		return AssistantResponse{}, mapDBError(err)
	}

	var reply string
	var card Card
	var trace []string

	// A correction of an existing transaction runs before intent routing, so
	// "ubah … jadi …" is never read as a new transaction or a budget proposal
	// (PRD §22.3, ISSUE-13).
	if corr := detectCorrection(text); corr.match {
		trace = []string{"correction"}
		reply, card, err = s.applyCorrection(ctx, q, userID, corr, wc)
	} else {
		intent, intentTrace := s.interpret(ctx, q, userID, text)
		trace = intentTrace

		// A reply to an outstanding clarification takes precedence over a
		// fresh intent: the draft is merged with this message, then either
		// saved or clarified again (PRD §7.4 / E2E #3).
		if draftReply, draftCard, handled, draftErr := s.tryResolveDraft(ctx, q, userID, conversation.ID, text, in.Source, wc); handled {
			reply, card, err = draftReply, draftCard, draftErr
			if err == nil {
				trace = append(trace, "draft_resolved")
			}
		} else {
			reply, card, err = s.routeIntent(ctx, q, userID, conversation.ID, text, intent, in.Source, wc)
		}
	}
	if err != nil {
		var domain *Error
		if errors.As(err, &domain) {
			card = Card{
				Type:  "error",
				Title: "Belum bisa diproses",
				Body:  domain.Message,
			}
			reply = domain.Message
			err = nil
		} else {
			return AssistantResponse{}, err
		}
	}

	cardJSON, err := json.Marshal(card)
	if err != nil {
		return AssistantResponse{}, err
	}
	traceJSON, err := json.Marshal(trace)
	if err != nil {
		return AssistantResponse{}, err
	}
	message, err := q.CreateAssistantMessage(ctx, generated.CreateAssistantMessageParams{
		ConversationID: conversation.ID,
		UserID:         pgUUIDValue(userID),
		Role:           "assistant",
		Content:        &reply,
		ToolCalls:      traceJSON,
		Card:           cardJSON,
	})
	if err != nil {
		return AssistantResponse{}, mapDBError(err)
	}
	if err := q.TouchConversation(ctx, generated.TouchConversationParams{
		ID:     conversation.ID,
		UserID: pgUUIDValue(userID),
	}); err != nil {
		return AssistantResponse{}, mapDBError(err)
	}

	return AssistantResponse{
		ConversationID: uuidString(conversation.ID),
		MessageID:      uuidString(message.ID),
		Reply:          reply,
		Card:           card,
		ToolTrace:      trace,
	}, nil
}

func (s *Service) ensureConversation(ctx context.Context, q *generated.Queries, userID uuid.UUID, id, text string) (generated.AssistantConversation, error) {
	if strings.TrimSpace(id) != "" {
		conversationID, err := parseUUID(id)
		if err != nil {
			return generated.AssistantConversation{}, err
		}
		conversation, err := q.GetConversationForUser(ctx, generated.GetConversationForUserParams{
			ID:     pgUUIDValue(conversationID),
			UserID: pgUUIDValue(userID),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return generated.AssistantConversation{}, NotFound("Percakapan")
		}
		if err != nil {
			return generated.AssistantConversation{}, mapDBError(err)
		}
		return conversation, nil
	}

	title := firstLine(text, 60)
	return q.CreateConversation(ctx, generated.CreateConversationParams{
		UserID: pgUUIDValue(userID),
		Title:  &title,
	})
}

// financialContext merangkum angka nyata dari backend untuk grounding jawaban
// konsultasi. Dipanggil dari interpret saat AI aktif; kegagalan query tidak
// menggagalkan pesan — konteks cukup dikosongkan (model lalu menyatakan angka
// tidak tersedia, sesuai Ai_intent.md §11).
func (s *Service) financialContext(ctx context.Context, q *generated.Queries, userID uuid.UUID) string {
	var b strings.Builder
	dash, err := s.Dashboard(ctx, q, userID, "this_month")
	if err != nil {
		return ""
	}
	fmt.Fprintf(&b, "Periode %s: pemasukan Rp%s, pengeluaran Rp%s, net Rp%s.",
		dash.Period.Label,
		formatRupiahPlain(dash.Summary.Income),
		formatRupiahPlain(dash.Summary.Expense),
		formatRupiahPlain(dash.Summary.Net),
	)
	// Hari ini selalu disertakan: pertanyaan harian ("berapa pengeluaran
	// hari ini") harus terjawab dari angka backend, termasuk saat nol.
	if today, err := s.Dashboard(ctx, q, userID, "today"); err == nil {
		fmt.Fprintf(&b, " Hari ini (%s): pemasukan Rp%s, pengeluaran Rp%s.",
			today.Period.Label,
			formatRupiahPlain(today.Summary.Income),
			formatRupiahPlain(today.Summary.Expense),
		)
	}
	if dash.Budget != nil {
		fmt.Fprintf(&b, " Budget %s: batas Rp%s, terpakai Rp%s, sisa Rp%s (%.0f%% terpakai).",
			dash.Period.Label,
			formatRupiahPlain(dash.Budget.OverallLimit),
			formatRupiahPlain(dash.Budget.Spent),
			formatRupiahPlain(dash.Budget.OverallLimit-dash.Budget.Spent),
			float64(dash.Budget.Spent)/float64(dash.Budget.OverallLimit)*100,
		)
	}
	if goals, err := s.ListSavingsGoals(ctx, q, userID); err == nil && len(goals) > 0 {
		var saved, target int64
		for _, g := range goals {
			saved += g.Saved
			target += g.TargetAmount
		}
		fmt.Fprintf(&b, " Tabungan: %d target, terkumpul Rp%s dari Rp%s.",
			len(goals), formatRupiahPlain(saved), formatRupiahPlain(target))
	}
	return b.String()
}

func (s *Service) interpret(ctx context.Context, q *generated.Queries, userID uuid.UUID, text string) (ai.Intent, []string) {
	trace := []string{"intent"}
	categories, _ := s.ListCategories(ctx, q, userID, "")
	accounts, _ := s.ListAccounts(ctx, q, userID)

	categoryNames := make([]string, 0, len(categories))
	for _, c := range categories {
		categoryNames = append(categoryNames, c.Name)
	}
	accountNames := make([]string, 0, len(accounts))
	for _, a := range accounts {
		accountNames = append(accountNames, a.Name)
	}

	if s.AI != nil && s.AI.Available() {
		// Backend-computed numbers for grounded consultation answers
		// (Ai_intent.md §5/§6) — diambil hanya saat AI aktif.
		financialContext := s.financialContext(ctx, q, userID)
		aiCtx, cancel := context.WithTimeout(ctx, s.Cfg.AITimeout)
		defer cancel()
		intent, err := s.AI.ExtractIntent(aiCtx, text, ai.Context{
			Categories: categoryNames,
			Accounts:   accountNames,
			Today:      s.NowUTC(),
			Financial:  financialContext,
		})
		if err == nil {
			trace = append(trace, "ai")
			if intent.Confidence <= 0 {
				intent.Confidence = 0.6
			}
			return intent, trace
		}
		trace = append(trace, "ai_fallback")
	}
	return heuristicIntent(text, s.Now()), trace
}

func (s *Service) routeIntent(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	conversationID pgtype.UUID,
	text string,
	intent ai.Intent,
	source string,
	wc WriteContext,
) (string, Card, error) {
	periodRaw := intent.Period
	if periodRaw == "" {
		periodRaw, _ = detectPeriodPhrase(strings.ToLower(text), s.Now().Year())
	}

	switch intent.Kind {
	case "create_transaction":
		return s.handleCreate(ctx, q, userID, conversationID, text, intent, source, wc)
	case "spending_summary":
		dashboard, err := s.Dashboard(ctx, q, userID, periodRaw)
		if err != nil {
			return "", Card{}, err
		}
		// Jawaban memakai angka hasil backend (Ai_intent.md §6/§11) dan
		// dibuat ramah, bukan sekadar konfirmasi proses. "Selisih" = net,
		// istilah yang sama dengan KPI Selisih di Beranda (bukan "Sisa"
		// yang terdengar seperti sisa budget).
		tail := "Selisih Rp" + formatRupiahPlain(dashboard.Summary.Net)
		if dashboard.Summary.Net < 0 {
			tail = "Defisit Rp" + formatRupiahPlain(-dashboard.Summary.Net)
		}
		body := fmt.Sprintf(
			"Periode %s: pemasukan Rp%s, pengeluaran Rp%s. %s. Rinciannya ada di kartu di bawah.",
			dashboard.Period.Label,
			formatRupiahPlain(dashboard.Summary.Income),
			formatRupiahPlain(dashboard.Summary.Expense),
			tail,
		)
		return body, Card{
			Type:       "financial_summary",
			Title:      "Ringkasan " + dashboard.Period.Label,
			Summary:    &dashboard.Summary,
			Period:     &dashboard.Period,
			Categories: dashboard.TopCategories,
			Trend:      dashboard.MonthlyTrend,
		}, nil
	case "largest_transactions":
		largest, err := s.LargestTransactions(ctx, q, userID, periodRaw, intent.Type, "", 5)
		if err != nil {
			return "", Card{}, err
		}
		body := "Ini transaksi terbesar pada " + largest.Period.Label + "."
		if len(largest.Items) == 0 {
			body = "Belum ada transaksi pada periode itu."
		}
		return body, Card{
			Type:   "largest_transaction",
			Title:  "Transaksi terbesar",
			Period: &largest.Period,
			Items:  largest.Items,
		}, nil
	case "unusual_spending":
		unusual, err := s.UnusualSpending(ctx, q, userID, periodRaw, "")
		if err != nil {
			return "", Card{}, err
		}
		body := "Tidak ada pengeluaran yang menyimpang pada " + unusual.Period.Label + "."
		switch {
		case !unusual.SufficientData:
			body = "Data belum cukup untuk analisis. " + strings.Join(unusual.Limitations, " ")
		case len(unusual.Findings) > 0:
			body = fmt.Sprintf("Ada %d kategori yang menyimpang dari pola %d bulan terakhir.",
				len(unusual.Findings), unusual.MonthsAnalyzed)
		}
		return body, Card{
			Type:    "unusual_spending",
			Title:   "Pengeluaran tidak biasa",
			Body:    body,
			Period:  &unusual.Period,
			Unusual: &unusual,
		}, nil
	case "search":
		queryText := intent.Query
		if queryText == "" {
			queryText = text
		}
		result, err := s.SearchTransactions(ctx, q, userID, queryText, 10)
		if err != nil {
			return "", Card{}, err
		}
		body := result.Filters.Interpreted
		if len(result.Items) == 0 {
			body += ". Tidak ada transaksi yang cocok."
		}
		return body, Card{
			Type:  "historical_search",
			Title: "Hasil pencarian",
			Body:  body,
			Items: result.Items,
		}, nil
	case "budget_status":
		r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
		if err != nil {
			return "", Card{}, err
		}
		budget, err := s.budgetForPeriod(ctx, q, userID, r)
		if err != nil {
			return "", Card{}, err
		}
		if budget == nil {
			return "Kamu belum punya budget untuk " + r.Label + ". Aku bisa bantu sarankan nominalnya.",
				Card{Type: "recommendation", Title: "Belum ada budget", Body: "Buat budget bulanan agar aku bisa memantau pengeluaranmu."}, nil
		}
		return "Sisa budget " + r.Label + " kamu ada di kartu berikut.",
			Card{Type: "budget_status", Title: "Budget " + r.Label, Budget: budget}, nil
	case "proposal":
		return s.handleBudgetProposal(ctx, q, userID, text, intent, periodRaw, wc)
	case "balance":
		return s.handleBalance(ctx, q, userID)
	case "delete_transaction":
		return s.handleDelete(ctx, q, userID, text, intent, periodRaw)
	case "consultation":
		// Jawaban grounding-nya dibangun model dari konteks finansial backend;
		// kosong berarti model tidak yakin — jatuh ke jawaban kapabilitas.
		if answer := strings.TrimSpace(intent.Answer); answer != "" {
			// Saran berbasis angka (withCard) ditemani kartu ringkasan:
			// metrik + chart dari backend, bukan angka karangan model.
			if intent.WithCard {
				if dashboard, err := s.Dashboard(ctx, q, userID, periodRaw); err == nil {
					return answer, Card{
						Type:       "financial_summary",
						Title:      "Ringkasan " + dashboard.Period.Label,
						Summary:    &dashboard.Summary,
						Period:     &dashboard.Period,
						Categories: dashboard.TopCategories,
						Trend:      dashboard.MonthlyTrend,
					}, nil
				}
			}
			return answer, Card{}, nil
		}
		fallthrough
	default:
		return "Aku bisa mencatat transaksi, menampilkan ringkasan, mencari riwayat, dan mengecek budget. Contoh: \"catat makan siang 25rb\".",
			Card{
				Type:  "recommendation",
				Title: "Mau mulai dari mana?",
				Body:  "Coba tulis: catat kopi 18rb, pengeluaran bulan ini, atau cari transaksi transportasi.",
			}, nil
	}
}

func (s *Service) handleCreate(ctx context.Context, q *generated.Queries, userID uuid.UUID, conversationID pgtype.UUID, text string, intent ai.Intent, source string, wc WriteContext) (string, Card, error) {
	categories, err := s.ListCategories(ctx, q, userID, "")
	if err != nil {
		return "", Card{}, err
	}

	input := CreateTransactionInput{
		Type:        strings.TrimSpace(intent.Type),
		Amount:      intent.Amount,
		Description: strings.TrimSpace(intent.Description),
		Source:      "chat",
	}
	if source == "voice" {
		input.Source = "voice"
	}
	if input.Type == "" {
		input.Type = "expense"
	}

	// One sentence, several transactions (plan.md §12.4, PRD §22.3). Every
	// segment must carry its own amount and description, otherwise the whole
	// sentence goes through the normal single-transaction path.
	if segments := s.buildMultiInputs(ctx, q, userID, text, input.Type, input.Source); segments != nil {
		if date := parseRelativeDate(text, s.Now(), s.location(ctx, q, userID)); date != nil {
			for i := range segments {
				segments[i].input.OccurredAt = date
			}
		}
		return s.createMultiple(ctx, q, userID, segments, categories, wc)
	}
	if input.Amount <= 0 {
		// Last resort: read an amount straight from the message.
		if amount, ok := parseAmountFromText(text); ok {
			input.Amount = amount
		}
	}
	// Day references ("kemarin", "tgl 15") set occurredAt in the user's zone
	// and never leak into the description.
	loc := s.location(ctx, q, userID)
	if date := parseRelativeDate(text, s.Now(), loc); date != nil {
		input.OccurredAt = date
		// The heuristic or model may already have filled the description, and
		// may have mangled the date phrase while doing so.
		input.Description = descriptionForDatedText(text, input.Description)
	}
	if input.Description == "" {
		input.Description = cleanDescription(stripDateWords(text))
	}

	missing := make([]string, 0, 2)
	if input.Amount <= 0 {
		missing = append(missing, "amount")
	}
	// A verb with no object ("bayar 50 ribu") would store a meaningless row,
	// so the assistant asks instead of guessing (PRD §7.4).
	if input.Description == "" || isGenericDescription(input.Description) {
		missing = append(missing, "description")
	}
	if len(missing) > 0 {
		return s.clarify(ctx, q, userID, conversationID, input, missing, "Aku butuh sedikit info lagi sebelum mencatat.")
	}

	// Category: prefer the mapped name, then a deterministic name match.
	var categoryID *string
	for _, category := range categories {
		if category.Kind != input.Type {
			continue
		}
		if intent.CategoryName != "" && strings.EqualFold(category.Name, intent.CategoryName) {
			id := category.ID
			categoryID = &id
			break
		}
		if categoryID == nil && stringMatchesQuery(strings.ToLower(category.Name), strings.ToLower(text)) {
			id := category.ID
			categoryID = &id
		}
	}
	input.CategoryID = categoryID

	// Account: match a named account, otherwise let the service resolve default.
	if intent.AccountName != "" {
		accounts, err := s.ListAccounts(ctx, q, userID)
		if err == nil {
			for _, account := range accounts {
				if strings.EqualFold(account.Name, intent.AccountName) {
					id := account.ID
					input.AccountID = &id
					break
				}
			}
		}
	}

	// Transfers need two distinct accounts. Ask instead of letting the service
	// reject the row with "akun asal dan tujuan harus berbeda" (PRD §7.4).
	if input.Type == "transfer" && (input.FromAccountID == nil || input.ToAccountID == nil) {
		accounts, err := s.ListAccounts(ctx, q, userID)
		if err != nil {
			return "", Card{}, err
		}
		input.FromAccountID, input.ToAccountID = resolveTransferAccounts(accounts, text, input.FromAccountID, input.ToAccountID)
		if input.FromAccountID == nil || input.ToAccountID == nil ||
			*input.FromAccountID == *input.ToAccountID {
			if len(accounts) < 2 {
				return "Untuk transfer kamu butuh dua akun yang berbeda. Tambah akun dulu di halaman Profil, lalu coba lagi.",
					Card{
						Type:  "recommendation",
						Title: "Butuh dua akun",
						Body:  "Transfer memerlukan akun asal dan akun tujuan yang berbeda. Kelola akun di halaman Profil.",
					}, nil
			}
			return s.clarify(ctx, q, userID, conversationID, input,
				[]string{"fromAccountId", "toAccountId"},
				"Sebutkan akun asal dan tujuan, misalnya: pindah 300rb dari BCA ke Cash.")
		}
	}

	tx, warnings, err := s.CreateTransaction(ctx, q, userID, input, wc)
	if err != nil {
		var domain *Error
		if errors.As(err, &domain) && domain.Code == CodeValidation {
			fields := []string{"accountId"}
			if strings.Contains(domain.Message, "Akun belum dipilih") {
				return s.clarify(ctx, q, userID, conversationID, input, fields, "Pilih akun dulu ya, karena kamu punya lebih dari satu akun.")
			}
		}
		return "", Card{}, err
	}

	title := "Tercatat: " + tx.Description
	reply := "Siap, " + tx.Description + " sebesar Rp" + formatRupiahPlain(tx.Amount) + " sudah dicatat."
	if len(warnings) > 0 {
		reply += " " + warnings[0].Message + "."
	}
	return reply, Card{
		Type:        "transaction_result",
		Title:       title,
		Body:        reply,
		Transaction: &tx,
		Warnings:    warnings,
	}, nil
}

func (s *Service) handleBudgetProposal(ctx context.Context, q *generated.Queries, userID uuid.UUID, text string, intent ai.Intent, periodRaw string, wc WriteContext) (string, Card, error) {
	r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
	if err != nil {
		return "", Card{}, err
	}
	budget, err := s.budgetForPeriod(ctx, q, userID, r)
	if err != nil {
		return "", Card{}, err
	}
	if budget == nil {
		return "Kamu belum punya budget aktif untuk " + r.Label + ".",
			Card{Type: "recommendation", Title: "Belum ada budget", Body: "Buat budget pertama di halaman Budget, lalu aku bisa mengusulkan perubahan."}, nil
	}

	amount := intent.Amount
	if amount <= 0 {
		if match := anyAmountRe.FindStringSubmatch(text); match != nil {
			if parsed, ok := parseRupiahNumber(match[1], match[2]); ok {
				amount = parsed
			}
		}
	}
	if amount <= 0 {
		return "Sebutkan nominal budget yang kamu inginkan, misalnya: ubah budget jadi 4jt.",
			Card{Type: "clarification", Title: "Nominal budget belum jelas", MissingFields: []string{"overallLimit"}, NeedsReply: true}, nil
	}

	proposal, err := s.CreateProposal(ctx, q, userID, CreateProposalInput{
		BudgetID:  budget.ID,
		Changes:   BudgetChanges{OverallLimit: &amount},
		Rationale: strPtr("Diusulkan lewat asisten"),
	}, wc)
	if err != nil {
		return "", Card{}, err
	}
	return "Aku sudah menyiapkan proposal budget. Perubahan baru berlaku setelah kamu setujui.",
		Card{Type: "budget_proposal", Title: "Usulan budget baru", Body: "Setujui untuk menerapkan.", Proposal: &proposal}, nil
}

// handleBalance menampilkan saldo per akun dan totalnya. Angka datang dari
// ListAccountBalances (ledger backend); balasan disusun server sebagai
// paragraf lalu poin "- " agar RichReply menyajikannya sebagai daftar rapi.
func (s *Service) handleBalance(ctx context.Context, q *generated.Queries, userID uuid.UUID) (string, Card, error) {
	accounts, err := s.ListAccounts(ctx, q, userID)
	if err != nil {
		return "", Card{}, err
	}
	if len(accounts) == 0 {
		return "Belum ada akun. Tambahkan akun dulu di halaman Profil agar saya bisa menampilkan saldo.",
			Card{Type: "recommendation", Title: "Belum ada akun", Body: "Kelola akun di halaman Profil."}, nil
	}
	var total int64
	for _, account := range accounts {
		total += account.Balance
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Total saldo kamu Rp%s dari %d akun.\n\n", formatRupiahPlain(total), len(accounts))
	for i, account := range accounts {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "- %s: Rp%s", account.Name, formatRupiahPlain(account.Balance))
	}
	return b.String(), Card{
		Type:     "account_balance",
		Title:    "Saldo akun",
		Body:     "Saldo = saldo awal plus mutasi transaksi aktif.",
		Accounts: accounts,
	}, nil
}

// handleDelete mencari kandidat transaksi yang dimaksud pengguna dan
// MENUNGGU konfirmasi lewat tombol di kartu; tidak ada baris yang dihapus
// di sini (Ai_intent.md: perubahan data butuh persetujuan pengguna).
func (s *Service) handleDelete(ctx context.Context, q *generated.Queries, userID uuid.UUID, text string, intent ai.Intent, periodRaw string) (string, Card, error) {
	queryText := strings.TrimSpace(intent.Query)
	if queryText == "" {
		queryText = text
	}
	candidates, filters, err := s.deleteCandidates(ctx, q, userID, queryText, periodRaw, intent.Amount, 5)
	if err != nil {
		return "", Card{}, err
	}
	switch {
	case len(candidates) == 0:
		return "Aku tidak menemukan transaksi aktif yang cocok. Sebutkan deskripsi atau nominalnya, misalnya: hapus transaksi makan siang 25rb.",
			Card{
				Type:  "recommendation",
				Title: "Tidak ada yang cocok",
				Body:  "Penghapusan selalu menampilkan kandidat dulu, tidak pernah langsung menghapus.",
			}, nil
	case len(candidates) == 1:
		tx := candidates[0]
		return "Ini transaksi yang kamu maksud. Periksa dulu, lalu tekan Hapus di kartu untuk membatalkannya.",
			Card{
				Type:        "delete_confirmation",
				Title:       "Hapus transaksi?",
				Body:        filters.Interpreted,
				Transaction: &tx,
			}, nil
	default:
		return fmt.Sprintf("Ada %d transaksi yang cocok. Sebutkan yang paling tepat berdasarkan nominal atau tanggalnya.", len(candidates)),
			Card{
				Type:  "historical_search",
				Title: "Kandidat transaksi",
				Body:  filters.Interpreted,
				Items: candidates,
			}, nil
	}
}

// deleteCandidates finds the active transactions a delete_transaction intent
// points at. Matching stays deterministic (Ai_intent.md: AI never picks the
// victim): structured filters first (period, type, amounts), then the
// leftover keyword as a whole phrase, then its longest word when the phrase
// is not a substring of any description. Voided rows never match because
// ListTransactions only returns status='active'.
func (s *Service) deleteCandidates(ctx context.Context, q *generated.Queries, userID uuid.UUID, raw, periodRaw string, amount int64, limit int) ([]TransactionDTO, SearchFilters, error) {
	filters, err := s.ParseSearch(ctx, q, userID, raw)
	if err != nil {
		return nil, SearchFilters{}, err
	}
	if filters.Period == "" && strings.TrimSpace(periodRaw) != "" {
		filters.Period = periodRaw
	}
	if amount > 0 {
		filters.MinAmount = &amount
		filters.MaxAmount = &amount
	}

	base := ListTransactionsInput{
		Type:       filters.Type,
		CategoryID: filters.CategoryID,
		Period:     filters.Period,
		Limit:      limit,
		MinAmount:  filters.MinAmount,
		MaxAmount:  filters.MaxAmount,
	}

	if filters.Text != "" {
		whole := base
		whole.Q = filters.Text
		page, err := s.ListTransactions(ctx, q, userID, whole)
		if err != nil {
			return nil, SearchFilters{}, err
		}
		if len(page.Items) > 0 {
			return page.Items, filters, nil
		}
		if word := longestWord(filters.Text); word != "" && word != filters.Text {
			byWord := base
			byWord.Q = word
			page, err := s.ListTransactions(ctx, q, userID, byWord)
			if err != nil {
				return nil, SearchFilters{}, err
			}
			if len(page.Items) > 0 {
				return page.Items, filters, nil
			}
		}
	}

	// No keyword, or nothing matched: the filtered recent list still gives
	// the user a concrete candidate set to pick from (or an honest empty).
	page, err := s.ListTransactions(ctx, q, userID, base)
	if err != nil {
		return nil, SearchFilters{}, err
	}
	return page.Items, filters, nil
}

// longestWord returns the longest whitespace-separated word (at least three
// runes) of a search phrase, used as the fallback delete-match keyword.
func longestWord(text string) string {
	best := ""
	for _, word := range strings.Fields(text) {
		if len([]rune(word)) < 3 {
			continue
		}
		if len([]rune(word)) > len([]rune(best)) {
			best = word
		}
	}
	return best
}

func (s *Service) clarify(ctx context.Context, q *generated.Queries, userID uuid.UUID, conversationID pgtype.UUID, input CreateTransactionInput, missing []string, body string) (string, Card, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return "", Card{}, err
	}
	draft, err := q.CreateDraft(ctx, generated.CreateDraftParams{
		UserID:         pgUUIDValue(userID),
		ConversationID: conversationID,
		Payload:        payload,
		MissingFields:  missing,
	})
	if err != nil {
		return "", Card{}, mapDBError(err)
	}
	return body, Card{
		Type:          "clarification",
		Title:         "Butuh konfirmasi",
		Body:          body,
		MissingFields: missing,
		DraftID:       uuidString(draft.ID),
		DraftData:     payload,
		NeedsReply:    true,
	}, nil
}

// tryResolveDraft continues an outstanding clarification. A message that
// clearly starts a new question/command is left to the normal intent router.
func (s *Service) tryResolveDraft(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	conversationID pgtype.UUID,
	text string,
	source string,
	wc WriteContext,
) (string, Card, bool, error) {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"berapa", "cari", "riwayat", "budget", "anggaran", "ringkasan", "total",
		"paling besar", "termahal", "terbesar", "tampilkan", "lihat daftar",
		// Pertanyaan/baru (cek saldo, hapus transaksi, cek transaksi) tidak
		// boleh tertelan draft pencatatan yang masih menunggu.
		"cek", "hapus", "saldo",
	} {
		if strings.Contains(lower, marker) {
			return "", Card{}, false, nil
		}
	}

	draft, err := q.GetPendingDraft(ctx, pgUUIDValue(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return "", Card{}, false, nil
	}
	if err != nil {
		return "", Card{}, false, mapDBError(err)
	}
	// A draft belongs to one thread; ignore it when the user switched topics.
	if draft.ConversationID.Valid && conversationID.Valid && draft.ConversationID != conversationID {
		return "", Card{}, false, nil
	}

	var payload CreateTransactionInput
	if err := json.Unmarshal(draft.Payload, &payload); err != nil {
		return "", Card{}, false, err
	}

	if payload.Amount <= 0 {
		if amount, ok := parseAmountFromText(text); ok {
			payload.Amount = amount
		}
	}
	// An empty or generic description ("bayar") is replaced by the reply.
	if payload.Description == "" || isGenericDescription(payload.Description) {
		// cleanDescription already drops the amount; strip date words first so
		// a reply like "tgl 15 oktober" is not half-eaten (ISSUE-14).
		if desc := cleanDescription(stripDateWords(text)); desc != "" && !isGenericDescription(desc) {
			payload.Description = desc
		}
	}
	if payload.AccountID == nil && containsField(draft.MissingFields, "accountId") {
		accounts, err := s.ListAccounts(ctx, q, userID)
		if err == nil {
			for _, account := range accounts {
				if strings.Contains(lower, strings.ToLower(account.Name)) {
					id := account.ID
					payload.AccountID = &id
					break
				}
			}
		}
	}
	if payload.Source == "" {
		payload.Source = "chat"
		if source == "voice" {
			payload.Source = "voice"
		}
	}

	// The reply may name the two transfer accounts ("dari BCA ke Cash").
	if payload.Type == "transfer" && (payload.FromAccountID == nil || payload.ToAccountID == nil) {
		accounts, err := s.ListAccounts(ctx, q, userID)
		if err == nil {
			payload.FromAccountID, payload.ToAccountID = resolveTransferAccounts(accounts, text, payload.FromAccountID, payload.ToAccountID)
		}
	}

	missing := make([]string, 0, 3)
	for _, field := range draft.MissingFields {
		switch field {
		case "amount":
			if payload.Amount <= 0 {
				missing = append(missing, field)
			}
		case "description":
			if payload.Description == "" || isGenericDescription(payload.Description) {
				missing = append(missing, field)
			}
		case "accountId":
			if payload.AccountID == nil {
				missing = append(missing, field)
			}
		case "fromAccountId":
			if payload.FromAccountID == nil {
				missing = append(missing, field)
			}
		case "toAccountId":
			if payload.ToAccountID == nil {
				missing = append(missing, field)
			}
		default:
			missing = append(missing, field)
		}
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", Card{}, true, err
	}

	if len(missing) > 0 {
		updated, err := q.UpdateDraft(ctx, generated.UpdateDraftParams{
			Payload:       encoded,
			MissingFields: missing,
			ID:            draft.ID,
			UserID:        pgUUIDValue(userID),
		})
		if err != nil {
			return "", Card{}, true, mapDBError(err)
		}
		body := "Aku butuh sedikit info lagi sebelum mencatat."
		return body, Card{
			Type:          "clarification",
			Title:         "Butuh konfirmasi",
			Body:          body,
			MissingFields: missing,
			DraftID:       uuidString(updated.ID),
			DraftData:     encoded,
			NeedsReply:    true,
		}, true, nil
	}

	tx, warnings, err := s.CreateTransaction(ctx, q, userID, payload, wc)
	if err != nil {
		return "", Card{}, true, err
	}
	if err := q.ResolveDraft(ctx, generated.ResolveDraftParams{
		ID:     draft.ID,
		UserID: pgUUIDValue(userID),
	}); err != nil {
		return "", Card{}, true, mapDBError(err)
	}

	reply := "Siap, " + tx.Description + " sebesar Rp" + formatRupiahPlain(tx.Amount) + " sudah dicatat."
	if len(warnings) > 0 {
		reply += " " + warnings[0].Message + "."
	}
	return reply, Card{
		Type:        "transaction_result",
		Title:       "Tercatat: " + tx.Description,
		Body:        reply,
		Transaction: &tx,
		Warnings:    warnings,
		DraftID:     uuidString(draft.ID),
	}, true, nil
}

func containsField(fields []string, wanted string) bool {
	for _, field := range fields {
		if field == wanted {
			return true
		}
	}
	return false
}

// resolveTransferAccounts fills the missing sides of a transfer from the
// sentence: "pindah 300rb dari BCA ke Cash" → from=BCA, to=Cash. It prefers
// the preposition that introduces each side, then falls back to naming any two
// distinct accounts in the reply. Equal accounts are discarded so the caller
// asks again instead of storing a broken row.
func resolveTransferAccounts(accounts []AccountDTO, text string, from, to *string) (*string, *string) {
	if from != nil && to != nil {
		return from, to
	}
	lower := strings.ToLower(text)

	if from == nil {
		from = accountMentionedAfter(accounts, lower, "dari ")
	}
	if to == nil {
		to = accountMentionedAfter(accounts, lower, "ke ")
	}

	if from == nil || to == nil {
		mentioned := make([]string, 0, len(accounts))
		for i := range accounts {
			name := strings.ToLower(accounts[i].Name)
			if name != "" && strings.Contains(lower, name) {
				mentioned = append(mentioned, accounts[i].ID)
			}
		}
		if from == nil && len(mentioned) > 0 {
			id := mentioned[0]
			from = &id
		}
		if to == nil {
			for _, id := range mentioned {
				if from == nil || id != *from {
					chosen := id
					to = &chosen
					break
				}
			}
		}
	}

	if from != nil && to != nil && *from == *to {
		return nil, nil
	}
	return from, to
}

// accountMentionedAfter returns the account named right after a preposition,
// cutting the phrase at the next preposition or conjunction.
func accountMentionedAfter(accounts []AccountDTO, lower, prefix string) *string {
	idx := strings.Index(lower, prefix)
	if idx < 0 {
		return nil
	}
	rest := lower[idx+len(prefix):]
	for _, stop := range []string{" ke ", " untuk ", " dan ", ",", "."} {
		if cut := strings.Index(rest, stop); cut >= 0 {
			rest = rest[:cut]
		}
	}
	rest = strings.TrimSpace(rest)
	for i := range accounts {
		name := strings.ToLower(accounts[i].Name)
		if name != "" && strings.Contains(rest, name) {
			id := accounts[i].ID
			return &id
		}
	}
	return nil
}

// ListConversations returns recent threads.
func (s *Service) ListConversations(ctx context.Context, q *generated.Queries, userID uuid.UUID, limit int) ([]ConversationDTO, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := q.ListConversations(ctx, generated.ListConversationsParams{
		UserID: pgUUIDValue(userID),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]ConversationDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, ConversationDTO{
			ID:            uuidString(row.ID),
			Title:         row.Title,
			CreatedAt:     timestamp(row.CreatedAt).Format(time.RFC3339),
			LastMessageAt: timestamp(row.LastMessageAt).Format(time.RFC3339),
		})
	}
	return out, nil
}

// GetConversation returns the message history of one thread.
func (s *Service) GetConversation(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string) ([]MessageDTO, error) {
	conversationID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	if _, err := q.GetConversationForUser(ctx, generated.GetConversationForUserParams{
		ID:     pgUUIDValue(conversationID),
		UserID: pgUUIDValue(userID),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFound("Percakapan")
		}
		return nil, mapDBError(err)
	}
	rows, err := q.ListAssistantMessages(ctx, generated.ListAssistantMessagesParams{
		ConversationID: pgUUIDValue(conversationID),
		UserID:         pgUUIDValue(userID),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]MessageDTO, 0, len(rows))
	for _, row := range rows {
		message := MessageDTO{
			ID:        uuidString(row.ID),
			Role:      row.Role,
			Content:   row.Content,
			CreatedAt: timestamp(row.CreatedAt).Format(time.RFC3339),
		}
		if len(row.Card) > 0 {
			message.Card = json.RawMessage(row.Card)
		}
		out = append(out, message)
	}
	return out, nil
}

// DeleteConversation removes a thread and its messages.
func (s *Service) DeleteConversation(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string) error {
	conversationID, err := parseUUID(id)
	if err != nil {
		return err
	}
	return mapDBError(q.DeleteConversation(ctx, generated.DeleteConversationParams{
		ID:     pgUUIDValue(conversationID),
		UserID: pgUUIDValue(userID),
	}))
}

// ---------------------------------------------------------------------------
// deterministic Indonesian intent extraction (fallback + guards)
// ---------------------------------------------------------------------------

func heuristicIntent(text string, now time.Time) ai.Intent {
	lower := strings.ToLower(text)
	intent := ai.Intent{Kind: "unknown", Confidence: 0.55}

	// Aksi atas data diprioritaskan agar tidak tertelan kata kerja lain:
	// "hapus transaksi beli pulsa" jangan jatuh ke create_transaction hanya
	// karena kata "beli"; "berapa saldo" jangan jadi spending_summary.
	switch {
	case strings.Contains(lower, "hapus"), strings.Contains(lower, "buang"),
		strings.Contains(lower, "hilangkan"), strings.Contains(lower, "batalkan"):
		intent.Kind = "delete_transaction"
		intent.Query = text
	case strings.Contains(lower, "saldo"):
		intent.Kind = "balance"
	default:
		createWords := []string{"catat", "beli", "bayar", "jajan", "makan", "minum", "terima", "gaji", "tambah", "abis", "keluar", "transfer", "nabung", "pindah"}
		for _, word := range createWords {
			if strings.Contains(lower, word) {
				intent.Kind = "create_transaction"
				break
			}
		}

		switch {
		case strings.Contains(lower, "paling besar"), strings.Contains(lower, "termahal"), strings.Contains(lower, "terbesar"):
			intent.Kind = "largest_transactions"
		case strings.Contains(lower, "tidak biasa"), strings.Contains(lower, "menyimpang"),
			strings.Contains(lower, "aneh"), strings.Contains(lower, "unusual"):
			intent.Kind = "unusual_spending"
		case strings.Contains(lower, "cari"), strings.Contains(lower, "riwayat"),
			strings.Contains(lower, "cek transaksi"), strings.Contains(lower, "lihat transaksi"),
			strings.Contains(lower, "daftar transaksi"):
			intent.Kind = "search"
		case strings.Contains(lower, "budget"), strings.Contains(lower, "anggaran"):
			if strings.Contains(lower, "ubah") || strings.Contains(lower, "ganti") ||
				strings.Contains(lower, "naik") || strings.Contains(lower, "turun") ||
				strings.Contains(lower, "buat") || strings.Contains(lower, "tambah") {
				intent.Kind = "proposal"
			} else {
				intent.Kind = "budget_status"
			}
		case strings.Contains(lower, "berapa"), strings.Contains(lower, "ringkasan"),
			strings.Contains(lower, "total"), strings.Contains(lower, "summary"),
			strings.Contains(lower, "pengeluaran"):
			intent.Kind = "spending_summary"
		}
		if intent.Kind == "create_transaction" && strings.Contains(lower, "berapa") {
			intent.Kind = "spending_summary"
		}
	}

	switch {
	case strings.Contains(lower, "terima"), strings.Contains(lower, "gaji"),
		strings.Contains(lower, "pendapatan"), strings.Contains(lower, "income"):
		intent.Type = "income"
	case strings.Contains(lower, "transfer"), strings.Contains(lower, "pindah"):
		intent.Type = "transfer"
	default:
		intent.Type = "expense"
	}

	if amount, ok := parseAmountFromText(text); ok {
		intent.Amount = amount
	}
	intent.Description = cleanDescription(text)
	intent.Period, _ = detectPeriodPhrase(lower, now.Year())
	return intent
}

func cleanDescription(text string) string {
	lower := strings.ToLower(text)
	// "tadi" is deliberately absent: stripDateWords removes the whole phrase
	// ("tadi pagi"), while dropping only "tadi" would leave "pagi" behind.
	for _, word := range []string{"tolong ", "coba ", "catat ", "tambahkan ", "tambah ", "aku ", "saya ", "barusan "} {
		lower = strings.ReplaceAll(lower, word, "")
	}
	lower = anyAmountRe.ReplaceAllString(lower, "")
	lower = strings.TrimSpace(lower)
	lower = strings.Join(strings.Fields(lower), " ")
	if len([]rune(lower)) > 120 {
		lower = string([]rune(lower)[:120])
	}
	if lower == "" {
		return ""
	}
	runes := []rune(lower)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// genericDescriptions are verbs with no object. Recording one would store a
// row such as "Bayar", which says nothing about what was paid, so the
// assistant asks instead of guessing (PRD §7.4 ambiguity → clarification).
var genericDescriptions = map[string]struct{}{
	"bayar": {}, "beli": {}, "belanja": {}, "belanjaan": {}, "jajan": {},
	"catat": {}, "terima": {}, "keluar": {}, "masuk": {}, "transfer": {},
	"pengeluaran": {}, "pemasukan": {}, "nabung": {}, "tambah": {},
}

// isGenericDescription reports whether a description is only a verb.
func isGenericDescription(description string) bool {
	normalized := strings.Trim(strings.ToLower(strings.TrimSpace(description)), ".!?")
	_, ok := genericDescriptions[normalized]
	return ok
}

func firstLine(text string, max int) string {
	line := strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
	if len([]rune(line)) > max {
		line = string([]rune(line)[:max])
	}
	return line
}

func strPtr(s string) *string { return &s }
