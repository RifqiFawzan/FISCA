# ISSUE-05 — Historical intelligence (M5)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. `GET /transactions/search?q=...` menerjemahkan frasa Indonesia menjadi filter terstruktur.
2. Frasa nominal/periode/superlatif tidak boleh bocor ke filter teks `q`.
3. `largest-transactions` berasal dari query DB dengan `ORDER BY amount DESC`.
4. Kategori yang cocok dihapus dari keyword sisa.

**Implementasi:** `internal/service/search.go` (`ParseSearch`, `leftoverText`, `detectPeriodPhrase`, `parseRupiahNumber`).

**QA:** `"pengeluaran bulan ini di atas 10rb"` → `period=this_month, type=expense, minAmount=10000, text=""`, hasil > 0; unit test `search_test.go` menguji nominal (`25rb`, `1,5jt`, `10.000`), periode, dan pembersihan keyword. → **PASS**
