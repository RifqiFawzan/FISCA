# Hasil Eval — Parser Deterministik FISCA

**Tanggal:** 9 November 2026
**Sumber kasus:** [`ai-cases.jsonl`](./ai-cases.jsonl) (plan.md §12.4, PRD §22.3)
**Runner:** `internal/service/eval_test.go` (`go test ./internal/service/ -run Eval -v`)

## Cara menjalankan

```bash
cd apps/api
go test ./internal/service/ -run Eval -count=1 -v
```

Runner tidak membutuhkan `OPENAI_API_KEY`, database, maupun jaringan: ia
menjalankan pipeline lokal (deteksi koreksi → heuristik intent → pemecah
multi-transaksi → parser tanggal → parser periode) terhadap setiap kasus.

## Cakupan set

| Kategori | Jumlah kasus |
|---|---|
| nominal | 6 |
| tanggal | 6 |
| multi-transaksi | 4 |
| code-switching | 4 |
| transfer-vs-expense | 4 |
| pertanyaan-vs-perintah | 4 |
| koreksi | 4 |
| ambiguitas | 4 |
| analitik | 4 |
| budget | 5 |
| **Total** | **45** (syarat plan.md §12.4: ≥40) |

Setiap baris memuat `id`, `kategori`, `input`, `ekspektasi`, dan `catatan`.
Field pada `ekspektasi` bersifat opsional: hanya field yang disebut yang dinilai.

## Metrik (run 9 November 2026)

| Metrik | Hasil |
|---|---|
| **Kasus lulus** | **45/45 (100,0%)** |
| intent accuracy | 45/45 (100,0%) |
| type accuracy (income/expense/transfer) | 5/5 (100,0%) |
| entity / nominal accuracy | 31/31 (100,0%) |
| multi-transaksi (jumlah baris) | 3/3 (100,0%) |
| tanggal transaksi | 6/6 (100,0%) |
| periode pencarian | 3/3 (100,0%) |
| clarification correctness | 5/5 (100,0%) |

**Ambang go/no-go (ditetapkan di `minEvalAccuracy`): 95%.**
Run pertama mencapai 100%, jadi ambang dipertahankan 95% sebagai pagar regresi:
penurunan ≥5 poin persentase akan menggagalkan suite.

## Temuan yang diperbaiki saat menyiapkan eval

Eval dirancangan lebih dulu dengan harapan PRD/plan, lalu parser disesuaikan —
bukan sebaliknya. Yang berubah:

1. **Nominal majemuk.** `"2 juta 500 ribu"` sebelumnya hanya membaca
   `2000000`. Ditambah `parseAmountFromText` yang menjumlahkan semua token
   bersatuan → `2500000`. Berlaku juga untuk pencarian (`"di atas 2 juta 500 ribu"`).
2. **Angka tanggal dianggap nominal.** `"bayar tgl 5 200rb"` dulu menjadi `5`.
   `parseAmountFromText` membuang frasa `tgl N …` lebih dulu → `200000`.
3. **`"pindah"` tidak dikenali sebagai transfer.** Masuk daftar kata kerja
   pencatatan + cabang tipe transfer.
4. **Ambiguitas langsung menyimpan.** `"bayar 50 ribu"` dulu membuat baris
   berdeskripsi "Bayar". Deskripsi berupa kata kerja tanpa objek kini memicu
   klarifikasi; balasan klarifikasi menggantikan deskripsi generik itu.
5. **Unusual spending tidak bisa dipanggil.** Frasa "tidak biasa/aneh/unusual"
   kini menghasilkan `kind=unusual_spending` dan kartu `unusual_spending`
   (dengan `sufficientData` + keterbatasan bila data kurang).
6. **`"buat budget …"` hanya menampilkan status.** Kini menjadi
   `budget_proposal` yang menunggu persetujuan, sesuai plan.md §12.4.

Temuan tambahan dari pemeriksaan live (bukan dari eval, karena runner tanpa
database): **transfer selalu berakhir error** `Akun asal dan tujuan harus
berbeda` ketika akun resolve ke default yang sama. Diperbaiki menjadi: <2 akun
→ kartu rekomendasi; ≥2 akun tanpa menyebut nama → klarifikasi
`fromAccountId`/`toAccountId`; balasan `"dari BCA ke Cash"` dicocokkan ke akun
nyata.

## Batasan

- **Jalur AI tidak ikut terukur.** Dengan `OPENAI_API_KEY` kosong yang diukur
  adalah fallback deterministik; eval terhadap model (transkrip, intent dari
  LLM) butuh key dan run terpisah, sesuai plan.md §12.4.
- **Runner tanpa database**, jadi asumsi: pengguna punya akun default dan
  kategori default. Kategori tidak dinilai (tidak ada field `ekspektasi.kategori`).
- **Belum dievaluasi:** koreksi multi-target, daftar tanpa kata penghubung,
  transfer antar-akun dalam satu kalimat, dan code-switching ekstrem
  (singkatan "2jt500rb" tanpa spasi).
- Metrik baru dihitung untuk parser; akurasi transkripsi STT dan angka hasil
  query database tidak dicakup di sini.
