package service

import "testing"

func TestSplitTransactionSegments(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"beli makan 25 ribu dan kopi 18 ribu", 2},
		{"beli makan 25rb, kopi 18rb", 2},
		{"beli makan 25rb; kopi 18rb; teh 5rb", 3},
		{"beli makan 25rb terus kopi 18rb", 2},
		{"catat makan siang 25rb", 1},
		{"  ", 0},
	}
	for _, tc := range cases {
		got := splitTransactionSegments(tc.text)
		if len(got) != tc.want {
			t.Errorf("splitTransactionSegments(%q) = %d parts, want %d (%v)", tc.text, len(got), tc.want, got)
		}
	}
}

func TestParseMultiSegmentsHappyPath(t *testing.T) {
	parsed := parseMultiSegments(
		splitTransactionSegments("beli makan 25 ribu dan kopi 18 ribu"),
		"expense",
		"chat",
	)
	if len(parsed) != 2 {
		t.Fatalf("segments = %d, want 2", len(parsed))
	}
	if parsed[0].input.Amount != 25000 || parsed[1].input.Amount != 18000 {
		t.Fatalf("amounts = %d, %d; want 25000, 18000", parsed[0].input.Amount, parsed[1].input.Amount)
	}
	if parsed[0].input.Description != "Beli makan" {
		t.Errorf("first description = %q, want %q", parsed[0].input.Description, "Beli makan")
	}
	if parsed[1].input.Description != "Kopi" {
		t.Errorf("second description = %q, want %q", parsed[1].input.Description, "Kopi")
	}
	for _, segment := range parsed {
		if segment.input.Type != "expense" || segment.input.Source != "chat" {
			t.Errorf("input = %+v, want type=expense source=chat", segment.input)
		}
	}
}

// A part without its own amount must not be dropped: the sentence goes back to
// the single path so nothing is silently lost.
func TestParseMultiSegmentsRequiresEveryAmount(t *testing.T) {
	cases := []string{
		"beli makan 25rb dan kopi",              // second part has no amount
		"beli makan dan kopi 18rb",              // first part has no amount
		"beli makan 25rb dan kopi dan teh 18rb", // middle part has no amount
	}
	for _, text := range cases {
		if got := parseMultiSegments(splitTransactionSegments(text), "expense", "chat"); got != nil {
			t.Errorf("parseMultiSegments(%q) = %d segments, want nil", text, len(got))
		}
	}
}

func TestParseMultiSegmentsRejectsTooFew(t *testing.T) {
	if got := parseMultiSegments(splitTransactionSegments("catat makan siang 25rb"), "expense", "chat"); got != nil {
		t.Fatalf("single sentence must not enter the multi path, got %d segments", len(got))
	}
	if got := parseMultiSegments(nil, "expense", "chat"); got != nil {
		t.Fatal("empty input must return nil")
	}
}

// Default type and source are applied per segment.
func TestParseMultiSegmentsDefaults(t *testing.T) {
	parsed := parseMultiSegments(splitTransactionSegments("beli makan 25rb dan kopi 18rb"), "", "")
	if len(parsed) != 2 {
		t.Fatalf("segments = %d, want 2", len(parsed))
	}
	for _, segment := range parsed {
		if segment.input.Type != "expense" || segment.input.Source != "chat" {
			t.Errorf("defaults not applied: %+v", segment.input)
		}
	}
}

func TestMatchCategoryID(t *testing.T) {
	categories := []CategoryDTO{
		{ID: "cat-1", Name: "Transportasi", Kind: "expense"},
		{ID: "cat-2", Name: "Gaji", Kind: "income"},
	}
	if got := matchCategoryID(categories, "expense", "transportasi ke kantor 15rb"); got != "cat-1" {
		t.Errorf("match = %q, want cat-1", got)
	}
	if got := matchCategoryID(categories, "expense", "beli gula 12rb"); got != "" {
		t.Errorf("match = %q, want empty", got)
	}
	if got := matchCategoryID(categories, "income", "gaji bulanan 5jt"); got != "cat-2" {
		t.Errorf("match = %q, want cat-2", got)
	}
}
