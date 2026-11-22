# ISSUE-14 — Tanggal & frasa lanjutan

**Status:** CLOSED (QA PASS)

**Sumber:** PRD §8 (contoh "kapan terakhir kali", "tgl 15 Oktober", "selama tiga bulan terakhir"), plan.md §12.4.

**Acceptance criteria**
1. `"minggu lalu"`, `"tadi pagi"`, `"bulan lalu"` sebagai tanggal transaksi dipetakan benar, tanpa mengubah perilaku `detectPeriodPhrase` di jalur pencarian.
2. `"tgl 15 Oktober"` membaca nama bulan, bukan hanya bulan berjalan.
3. Frasa rentang pada pencarian: `"selama tiga bulan terakhir"`, `"3 bulan terakhir"` → filter periode yang benar.
4. Semua kasus punya unit test; jalur lama (pencarian natural, tanggal relatif ISSUE-09) tidak berubah hasilnya.

**Implementasi**
- `internal/service/dates.go`: `parseRelativeDate` menambah "minggu lalu" (−7 hari), "tadi pagi/siang/sore/malam" (hari ini), "bulan lalu" (tanggal sama bulan sebelumnya, `AddDate` menyesuaikan bulan pendek), dan `tgl N NamaBulan` dengan rollover ke tahun sebelumnya bila masa depan. `dayOfMonthRe` kini menangkap nama bulan; `monthByName` memuat nama penuh + singkatan (jan…des). `stripDateWords` ikut membuang frasa baru.
- `internal/service/search.go`: `detectPeriodPhrase` mengenali `"N bulan terakhir"` / `"… tiga bulan terakhir"` → kunci periode `Nm`, dengan `parseRollingMonths` (angka digit + kata: sebulan…dua belas).
- `internal/period/period.go`: `Parse` menerima `Nm` (1–60) sebagai jendela menggulir yang **memuat hari ini** (`[hari ini − N bulan, besok)`), label "N bulan terakhir", bucket `month` untuk N≥3.
- `descriptionForDatedText`: mempertahankan deskripsi dari intent bila masih utuh, dan mengambil ulang dari kalimat mentah bila frasa tanggal sudah setengah hancur.

**QA**
- Unit: `TestParseRelativeDate` (+7 kasus: minggu lalu, tadi pagi, bulan lalu, nama bulan, rollover tahun, nama bulan tak dikenal), `TestCleanDescriptionDropsDateWords` (+4), `TestDescriptionForDatedText` (6), `TestDetectPeriodPhrase` (+4 kasus `Nm`), `TestDetectedPeriodParses` (key hasil deteksi wajib diterima `period.Parse`), `TestParseRollingMonths` (jendela, label, bucket, penolakan `0m`/`99m`).
- Regresi: `go test ./... -count=1` semua paket `ok` dengan `TEST_DATABASE_URL`; `go vet` bersih; `npm run smoke:api` lulus.
- Live di Docker API: `catat kopi tadi pagi 18rb` → 2026-11-08T05:00:00Z desc `Kopi`; `catat makan minggu lalu 25rb` → 2026-11-01 desc `Makan`; `catat beli buku bulan lalu 40rb` → 2026-10-08 desc `Beli buku`; `bayar listrik tgl 15 oktober 200rb` → 2026-10-15 desc `Bayar listrik`.
- Regresi pencarian: `"pengeluaran selama tiga bulan terakhir"` → `period=3m` + label "3 bulan terakhir"; `"pengeluaran bulan lalu"` → `last_month` (tidak berubah); `"pengeluaran minggu lalu"` → `7d` (tidak berubah).

**Temuan QA selama pengerjaan**
- Bug 1: `"tgl 15 oktober"` menghasilkan deskripsi `Bayar listrik tgl oktober`. Penyebab: heuristik membuang angka mentah (`15`) lebih dulu, sehingga `stripDateWords` tidak lagi mengenali `tgl 15 oktober`. Diperbaiki dengan `descriptionForDatedText` yang mengambil ulang dari kalimat mentah bila hasilnya masih memuat `tgl`.
- Bug 2: `"catat kopi tadi pagi 18rb"` menghasilkan `Kopi pagi`. Penyebab: daftar pembuang kata di `cleanDescription` memuat `"tadi "`, jadi hanya "tadi" yang terbuang dan "pagi" tersisa. Dihapus dari daftar itu — `stripDateWords` kini membuang frasa `tadi …` sebagai satu unit.
- Keduanya dijadikan test regresi (`TestDescriptionForDatedText` dan `TestCleanDescription`).

**Batasan:** "bulan lalu" sebagai tanggal memakai tanggal yang sama di bulan sebelumnya (pendekatan, bukan tanggal pasti); "minggu lalu" di jalur **pencarian** tetap `7d` (periode menggulir 7 hari) sesuai perilaku lama, bukan minggu kalender.
