package service

import (
	"testing"
)

func TestDetectCorrection(t *testing.T) {
	amount := func(v int64) *int64 { return &v }

	cases := []struct {
		name       string
		text       string
		wantMatch  bool
		wantTarget string
		wantAmount *int64
		wantDesc   string
	}{
		{
			name: "ubah nominal", text: "ubah kopi jadi 15rb",
			wantMatch: true, wantTarget: "kopi", wantAmount: amount(15000),
		},
		{
			name: "ubah deskripsi", text: "ganti kopi jadi teh",
			wantMatch: true, wantTarget: "kopi", wantDesc: "Teh",
		},
		{
			name: "ubah nominal dan deskripsi", text: "edit teh ke 12rb",
			wantMatch: true, wantTarget: "teh", wantAmount: amount(12000),
		},
		{
			name: "koreksi dengan tanggal", text: "koreksi beli makan kemarin jadi 30rb",
			wantMatch: true, wantTarget: "beli makan", wantAmount: amount(30000),
		},
		{
			name: "seharusnya tanpa target", text: "salah, seharusnya 15rb",
			wantMatch: true, wantTarget: "", wantAmount: amount(15000),
		},
		// Sentence types that must keep their own handlers.
		{name: "budget", text: "ubah budget jadi 2jt"},
		{name: "kategori", text: "ganti kategori jadi makanan"},
		{name: "akun", text: "ubah akun jadi BCA"},
		{name: "pertanyaan", text: "berapa pengeluaran bulan ini"},
		{name: "pencatatan baru", text: "catat kopi 18rb"},
		{name: "bukan koreksi", text: "lihat riwayat"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectCorrection(tc.text)
			if got.match != tc.wantMatch {
				t.Fatalf("match = %v, want %v", got.match, tc.wantMatch)
			}
			if !tc.wantMatch {
				return
			}
			if got.target != tc.wantTarget {
				t.Errorf("target = %q, want %q", got.target, tc.wantTarget)
			}
			switch {
			case tc.wantAmount == nil && got.amount != nil:
				t.Errorf("amount = %d, want none", *got.amount)
			case tc.wantAmount != nil && got.amount == nil:
				t.Errorf("amount = none, want %d", *tc.wantAmount)
			case tc.wantAmount != nil && *got.amount != *tc.wantAmount:
				t.Errorf("amount = %d, want %d", *got.amount, *tc.wantAmount)
			}
			if got.description != tc.wantDesc {
				t.Errorf("description = %q, want %q", got.description, tc.wantDesc)
			}
		})
	}
}

func TestParseCorrectionValue(t *testing.T) {
	amount, desc := parseCorrectionValue("teh 15rb")
	if amount == nil || *amount != 15000 {
		t.Errorf("amount = %v, want 15000", amount)
	}
	if desc != "Teh" {
		t.Errorf("desc = %q, want %q", desc, "Teh")
	}

	amount, desc = parseCorrectionValue("1,5 jt")
	if amount == nil || *amount != 1500000 {
		t.Errorf("amount = %v, want 1500000", amount)
	}
	if desc != "" {
		t.Errorf("desc = %q, want empty", desc)
	}
}

func TestFindCorrectionTarget(t *testing.T) {
	items := []TransactionDTO{
		{ID: "newest", Description: "Kopi susu"},
		{ID: "older", Description: "Beli makan siang"},
	}

	if got := findCorrectionTarget(items, ""); got == nil || got.ID != "newest" {
		t.Errorf("empty target picked %v, want newest", got)
	}
	if got := findCorrectionTarget(items, "makan"); got == nil || got.ID != "older" {
		t.Errorf("fragment target picked %v, want older", got)
	}
	if got := findCorrectionTarget(items, "sepatu"); got != nil {
		t.Errorf("unmatched fragment picked %v, want nil", got)
	}
	if got := findCorrectionTarget(nil, ""); got != nil {
		t.Errorf("no items picked %v, want nil", got)
	}
}
