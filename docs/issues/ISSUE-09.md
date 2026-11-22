# ISSUE-09 — Tanggal relatif di asisten

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. "kemarin" menghasilkan `occurredAt` hari sebelumnya; "kemarin lusa" dua hari sebelumnya; "hari ini" hari ini; ketiganya dihitung di zona waktu pengguna.
2. "tgl N" memilih tanggal N di bulan berjalan; bila N di masa depan, memakai bulan sebelumnya.
3. Tanggal tidak valid ("tgl 45", "tgl 31" di bulan 30 hari) tidak mengubah tanggal; server memakai waktu sekarang.
4. Kata penanda tanggal tidak ikut tersimpan di deskripsi.
5. Waktu transaksi diset ke pukul 12:00 lokal agar tidak bergeser hari saat konversi zona.

**Implementasi:** `internal/service/dates.go` (`parseRelativeDate`, `stripDateWords`), dipanggil dari `handleCreate` di `internal/service/assistant.go`.

**QA:**
- Unit test `dates_test.go`: 8 kasus tanggal (termasuk batas masa depan dan bulan pendek) + regresi deskripsi dari heuristik.
- Jalur live: "catat makan siang kemarin 25rb" → deskripsi `Makan siang`, `occurredAt=2026-11-07T05:00:00Z` (= 7 Nov 12:00 WIB).
- Regresi: `npm run smoke:api` lulus penuh; isolasi lintas-user tetap `otherCount=0`.
- Ditemukan saat QA: deskripsi tetap berisi "kemarin" karena heuristik sudah mengisi deskripsi lebih dulu. Diperbaiki dan dijadikan test regresi.

**Batasan:** "tadi pagi" dan "minggu lalu" belum dipetakan; tanggal absolut hanya untuk format "tgl N" di bulan berjalan.
