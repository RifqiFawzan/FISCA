# ISSUE-13 — Koreksi transaksi lewat chat

**Status:** CLOSED (QA PASS)

**Sumber:** PRD §22.3 "Koreksi transaksi sebelumnya", plan.md §12.4 (contoh: `"ubah kopi tadi jadi 15 rb"` → transaksi terakhir dikoreksi).

**Acceptance criteria**
1. Kalimat koreksi (`ubah … jadi …`, `ganti … ke …`, `salah, seharusnya …`, `koreksi …`) mengubah transaksi yang dimaksud, bukan membuat baru.
2. Target koreksi ditentukan dari deskripsi (terbaru lebih dulu); bila tidak ada yang cocok → klarifikasi, bukan transaksi baru.
3. Perubahan memakai jalur `UpdateTransaction` dengan `version`, sehingga audit trail terisi dan kunci optimistik berlaku.
4. Kalimat budget/kategori/akun/pertanyaan tidak tertangkap detektor koreksi.
5. Unit test detektor + integrasi test alur koreksi, audit, kegagalan, dan regresi budget.

**Implementasi**
- `internal/service/correction.go`: `detectCorrection` (pola `ubah|ganti|rubah|koreksi|edit … jadi|ke …` dan `salah|maaf|koreksi … seharusnya …`), `correctionStopwords`, `parseCorrectionValue`, `stripFiller`, `findCorrectionTarget`, `applyCorrection`.
- Dicabang paling awal di `HandleMessage` (sebelum interpret/draft) supaya `ubah … jadi …` tidak pernah terbaca sebagai pencatatan baru atau proposal budget; `trace` memuat `correction`.

**QA**
- Unit `correction_test.go`: 11 kasus detektor (5 koreksi valid + 6 kalimat yang harus jatuh ke handler lain), parsing nominal/deskripsi, pemilihan target.
- Integrasi: `TestChatCorrectionUpdatesTransactionAndAudits` (id sama, `amount` 18000→15000, `version` naik, jumlah baris tetap 1, saldo bergerak tepat, `audit_logs.action='update'` terisi), `TestChatCorrectionWithoutMatchClarifies` (klarifikasi, nol baris, saldo tidak berubah), `TestBudgetCorrectionStillCreatesProposal` (regresi: `budget_proposal`).
- Live di Docker API: `catat beli roti 12rb` → `ubah roti jadi 10rb` menghasilkan **id sama**, `amount=10000`, `version=2`, `trace=correction`; `ubah sepatu jadi 500rb` → klarifikasi; `ubah budget jadi 3jt` → `budget_proposal`.
- `go vet` bersih; `go test ./... -count=1` semua paket `ok`; `npm run smoke:api` lulus.

**Temuan QA selama pengerjaan**
- Assertion saldo saya kebalik: menurunkan pengeluaran 18.000→15.000 **menaikkan** saldo 3.000 (bukan menurunkannya). Perilaku sistem benar, test yang diperbaiki.

**Batasan:** bentuk kalimat lain belum didukung — `"ubah jadi 15rb"` tanpa objek, koreksi akun/kategori (ditangani handler masing-masing), dan koreksi transaksi di luar 25 baris terakhir.
