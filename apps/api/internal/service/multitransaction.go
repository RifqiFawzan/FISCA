package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// segmentSplitRe separates the items of one sentence. Indonesian writers use
// conjunctions as often as commas, so both are treated as separators.
var segmentSplitRe = regexp.MustCompile(`\s*(?:,|;|\bdan\b|\bterus\b|\blalu\b|\bplus\b)\s*`)

// multiSegment is one parsed item of a multi-transaction sentence.
type multiSegment struct {
	phrase string
	input  CreateTransactionInput
}

// splitTransactionSegments breaks a sentence into candidate item phrases.
func splitTransactionSegments(text string) []string {
	parts := segmentSplitRe.Split(strings.ToLower(text), -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// parseMultiSegments keeps the split only when every part carries its own
// amount and description. Anything less falls back to the single-transaction
// path, so no part of the sentence is silently dropped.
func parseMultiSegments(segments []string, intentType, source string) []multiSegment {
	if len(segments) < 2 {
		return nil
	}
	out := make([]multiSegment, 0, len(segments))
	for _, phrase := range segments {
		amount, ok := parseAmountFromText(phrase)
		if !ok {
			return nil
		}
		// Date words first: stripping amounts first can mangle "tgl 15 oktober"
		// into something stripDateWords no longer recognises.
		description := cleanDescription(stripDateWords(phrase))
		if description == "" {
			return nil
		}
		input := CreateTransactionInput{
			Type:        intentType,
			Amount:      amount,
			Description: description,
			Source:      source,
		}
		if input.Type == "" {
			input.Type = "expense"
		}
		if input.Source == "" {
			input.Source = "chat"
		}
		out = append(out, multiSegment{phrase: phrase, input: input})
	}
	return out
}

// buildMultiInputs decides whether a message should be recorded as several
// transactions. It refuses transfers (each needs from/to) and messages without
// a default account to fall back on, leaving those to the normal path.
func (s *Service) buildMultiInputs(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	text string,
	intentType string,
	source string,
) []multiSegment {
	if intentType == "transfer" {
		return nil
	}
	parsed := parseMultiSegments(splitTransactionSegments(text), intentType, source)
	if parsed == nil {
		return nil
	}
	accounts, err := s.ListAccounts(ctx, q, userID)
	if err != nil {
		return nil
	}
	for _, account := range accounts {
		if account.IsDefault {
			for i := range parsed {
				id := account.ID
				parsed[i].input.AccountID = &id
			}
			return parsed
		}
	}
	// More than one account and no default: let the single path ask which one.
	return nil
}

// createMultiple records every segment inside the caller's transaction, so a
// failure rolls the whole sentence back instead of leaving it half recorded.
func (s *Service) createMultiple(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	segments []multiSegment,
	categories []CategoryDTO,
	wc WriteContext,
) (string, Card, error) {
	created := make([]TransactionDTO, 0, len(segments))
	warnings := make([]Warning, 0, 2)

	for i := range segments {
		segment := &segments[i]
		if id := matchCategoryID(categories, segment.input.Type, segment.phrase); id != "" {
			segment.input.CategoryID = &id
		}
		tx, txWarnings, err := s.CreateTransaction(ctx, q, userID, segment.input, wc)
		if err != nil {
			return "", Card{}, err
		}
		created = append(created, tx)
		warnings = append(warnings, txWarnings...)
	}

	reply := fmt.Sprintf("Siap, %d transaksi dicatat:", len(created))
	for _, tx := range created {
		reply += fmt.Sprintf(" %s Rp%s,", tx.Description, formatRupiahPlain(tx.Amount))
	}
	reply = strings.TrimSuffix(reply, ",") + "."
	if len(warnings) > 0 {
		reply += " " + warnings[0].Message + "."
	}

	return reply, Card{
		Type:        "transaction_result",
		Title:       fmt.Sprintf("%d transaksi dicatat", len(created)),
		Body:        reply,
		Transaction: &created[0],
		Items:       created,
		Warnings:    warnings,
	}, nil
}

// matchCategoryID finds a category whose name appears in the phrase.
func matchCategoryID(categories []CategoryDTO, kind, phrase string) string {
	for _, category := range categories {
		if category.Kind != kind {
			continue
		}
		if stringMatchesQuery(strings.ToLower(category.Name), strings.ToLower(phrase)) {
			return category.ID
		}
	}
	return ""
}
