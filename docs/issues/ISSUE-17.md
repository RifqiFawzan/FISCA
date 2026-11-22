# ISSUE-17 — Eval set AI 40 kasus Bahasa Indonesia

**Status:** CLOSED (QA PASS)

**Sumber:** plan.md §12.4, PRD §22.3.

**Acceptance criteria**
1. `docs/eval/ai-cases.jsonl` berisi ≥40 kasus dengan kategori: nominal, tanggal, multi-transaksi, code-switching, transfer vs expense, pertanyaan vs perintah, koreksi, ambiguitas, analitik, budget.
2. Setiap baris memuat `input`, `kategori`, `ekspektasi` (bentuk terstruktur), dan `catatan`.
3. Runner test membaca file itu dan menguji parser/heuristik secara lokal tanpa panggilan model bila key kosong.
4. Metrik dihitung: intent accuracy, entity/nominal accuracy, duplicate rate, clarification correctness.
5. Hasil run pertama ditulis ke `docs/eval/hasil.md` beserta ambang go/no-go yang disetujui.

**Implementasi**
- `docs/eval/ai-cases.jsonl`: **45 kasus** dalam 10 kategori (syarat ≥40), tiap baris `id/kategori/input/ekspektasi/catatan`.
- `internal/service/eval_test.go`: `TestEvalSetCoversPlannedCategories` (jumlah + kategori + kelengkapan field) dan `TestEvalDeterministicAccuracy` (menjalankan pipeline lokal terhadap setiap kasus, menghitung metrik per dimensi, lalu membandingkan dengan ambang `minEvalAccuracy = 95%`).
- `docs/eval/hasil.md`: metrik run pertama + ambang go/no-go + temuan + batasan.

**Perbaikan yang lahir dari eval** (ekspektasi ditulis dulu mengikuti PRD/plan, parser disesuaikan — bukan sebaliknya):
1. Nominal majemuk `2 juta 500 ribu` → `2500000` lewat `parseAmountFromText` (menjumlahkan token bersatuan; berlaku juga untuk pencarian).
2. Angka tanggal tidak lagi dianggap nominal: `bayar tgl 5 200rb` → `200000` (bukan `5`).
3. `"pindah"` dikenali sebagai transfer (kata kerja + cabang tipe).
4. Deskripsi generik (`"bayar 50 ribu"`) memicu klarifikasi, dan balasan klarifikasi menggantikannya — bukan menyimpan baris "Bayar".
5. Frasa "tidak biasa/aneh/unusual" → `kind=unusual_spending` + kartu `unusual_spending` (belum pernah bisa dipanggil sebelumnya).
6. `"buat budget …"` → `budget_proposal` menunggu persetujuan (sebelumnya hanya status).
7. **Temuan live:** transfer selalu error `akun asal dan tujuan harus berbeda`. Kini <2 akun → kartu rekomendasi; ≥2 akun tanpa menyebut nama → klarifikasi `fromAccountId`/`toAccountId`; balasan `"dari BCA ke Cash"` dicocokkan ke akun nyata.

**QA**
- `go test ./internal/service/ -run Eval -v`: **45/45 kasus lulus (100%)** — intent 45/45, type 5/5, nominal 31/31, multi 3/3, tanggal 6/6, periode 3/3, klarifikasi 5/5; di atas ambang 95%.
- `go vet ./...` dan `go test ./... -count=1` (dengan `TEST_DATABASE_URL`) bersih — tidak ada regresi dari perbaikan parser.
- Live di Docker API: `catat bayar kos 2 juta 500 ribu` → `amount=2500000`; `bayar 50 ribu` → `clarification missing=description`; `apakah ada pengeluaran tidak biasa bulan ini` → `unusual_spending`; `buat budget makanan 1 juta` → `budget_proposal`; transfer 1 akun → rekomendasi, tanpa nama → klarifikasi, balasan `dari BCA ke Cash` → `transaction_result from=BCA to=Cash amount=300000`.
- Regresi: `npm run smoke:api` lulus (`otherCount=0`); alur klarifikasi draft lama tetap `clarification` → `transaction_result`.
- Rebuild stack Docker: `fisca-api`/`fisca-web` healthy.

**Batasan:** yang terukur adalah fallback deterministik (tanpa `OPENAI_API_KEY`); eval terhadap model dan transkripsi STT butuh key + run terpisah. Runner tanpa database sehingga mengasumsikan akun/kategori default; kategori tidak dinilai. Belum dievaluasi: koreksi multi-target, daftar tanpa kata penghubung, dan singkatan tanpa spasi (`2jt500rb`).
