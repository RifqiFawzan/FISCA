# ISSUE-12 — Multi-transaksi dalam satu kalimat

**Status:** CLOSED (QA PASS)

**Sumber:** PRD §22.3 ("Multi-transaction dalam satu kalimat"), plan.md §12.4 (ekspektasi: `"beli makan 25 ribu dan kopi 18 ribu"` → 2 transaksi).

**Acceptance criteria**
1. Satu kalimat berisi beberapa item dengan nominal masing-masing menghasilkan beberapa baris transaksi, bukan satu.
2. Nominal per item dibaca dengan notasi Indonesia (`25 ribu`, `18rb`, `1,5jt`).
3. Deskripsi per item dibersihkan dari nominal dan penanda tanggal.
4. Seluruh item dicatat dalam satu transaksi database: bila satu gagal, tidak ada yang tersimpan.
5. Kalimat yang salah satu itemnya tanpa nominal tidak dipecah — jatuh ke jalur tunggal agar tidak ada bagian yang hilang diam-diam.
6. Transfer tidak masuk jalur ini (butuh akun asal/tujuan).
7. Tanpa akun default, kembali ke jalur tunggal yang meminta klarifikasi akun.
8. Kartu tetap memakai `card.type = transaction_result` dan menyertakan `items` berisi semua transaksi.

**Implementasi**
- `internal/service/multitransaction.go`: `splitTransactionSegments` (pemisah `, ; dan terus lalu plus`), `parseMultiSegments` (semua segmen wajib punya nominal + deskripsi), `buildMultiInputs` (penjaga tipe transfer & akun default), `createMultiple` (pencatatan berurutan dalam transaksi pemanggil), `matchCategoryID`.
- Dicabang dari `handleCreate` di `internal/service/assistant.go`.

**QA**
- Unit: `multitransaction_test.go` — pemisah, jalur bahagia (25.000/18.000, `Beli makan`/`Kopi`), 3 kasus fallback ketika satu item tanpa nominal, penolakan kalimat tunggal, default tipe/source, pencocokan kategori.
- Integrasi: `TestMultiTransactionSentenceCreatesTwoRows` (2 baris, saldo turun tepat 43.000) dan `TestMultiTransactionFallsBackWhenAmountMissing` (1 baris, saldo turun 25.000).
- Live di Docker API: `beli makan 25 ribu dan kopi 18 ribu` → kartu `2 transaksi dicatat`, item `Beli makan 25000` dan `Kopi 18000`.
- `go test ./... -count=1` dengan `TEST_DATABASE_URL`: semua paket `ok`; `go vet` bersih; `npm run smoke:api` lulus.

**Temuan QA selama pengerjaan**
- Helper awal tidak menyetel `source` bila kosong (jalur nyata selalu mengirim `chat`/`voice`, tetapi helper jadi tidak mandiri). Diperbaiki.
- Kasus uji "fragment tanpa nominal di akhir kalimat" ternyata tidak menghasilkan fragment sama (kata pemisah di ujung kalimat terbuang). Diganti dengan `"… dan kopi dan teh 18rb"` yang benar-benar punya segmen tanpa nominal.

**Batasan:** belum mendukung daftar tanpa kata penghubung (`beli makan 25rb kopi 18rb`), item transfer dalam satu kalimat, dan urutan item yang berbeda dari urutan penulisan.
