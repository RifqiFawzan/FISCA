# ISSUE-10 — Integration & security test otomatis

**Status:** CLOSED (QA PASS)

**Sumber:** PRD §23 exit criteria #19 ("Security test membuktikan tidak ada akses lintas-user") dan #20 ("Core flows lulus integration dan end-to-end tests"). Sebelumnya hanya ada smoke script manual.

**Acceptance criteria**
1. User B tidak dapat membaca, mengubah, membatalkan, atau menyetujui data milik User A: transaksi, budget, status budget, proposal. Respons harus 404.
2. Daftar transaksi dan pencarian User B tidak memuat data User A.
3. Retry dengan `Idempotency-Key` yang sama tidak menghasilkan transaksi kedua (exit #8).
4. Key yang sama dengan body berbeda menghasilkan `409 IDEMPOTENCY_MISMATCH`.
5. Transaksi melewati batas budget tetap tersimpan dengan peringatan (exit #16).
6. Perubahan budget tidak berlaku sebelum disetujui; `baseVersion` basi ditolak `409 BUDGET_STALE`; persetujuan menulis audit log (exit #14, #15).
7. Pesan ambigu menjadi draft yang tidak mengubah saldo; balasan klarifikasi menyelesaikannya (exit #6, #7).
8. "Terbesar" sama dengan hasil query DB; pertanyaan unusual spending menyatakan keterbatasan data bila tidak cukup (exit #11, #12).

**Implementasi**
- `internal/server/integration_test.go`: 9 test melalui router asli dan database asli. Setiap test memakai user acak yang dihapus saat cleanup (semua tabel user-owned `ON DELETE CASCADE`).
- Dijalankan hanya bila `TEST_DATABASE_URL` diset; tanpa variabel itu test di-skip, bukan gagal.
- Perbaikan kode dari temuan test: `RejectProposal` oleh user lain sebelumnya mengembalikan `409 CONFLICT`; sekarang `404 NOT_FOUND` bila proposal bukan milik user, dan `409` hanya bila proposal milik user tetapi sudah diputuskan. Ini mencegah pemeriksaan kepemilikan memberi jawaban yang berbeda dari "tidak ada".

**Temuan QA selama pengerjaan**
- Test over-budget awalnya terlalu ketat (hanya menerima `BUDGET_WARNING`). Sistem memakai `BUDGET_EXCEEDED` bila batas sudah terlampaui; keduanya peringatan yang sah. Test diperbaiki; perilaku tidak berubah.
- Kode `RejectProposal` yang menyebabkan 409 untuk akses lintas-user (lihat di atas).

**QA**
- `go test ./... -count=1` dengan `TEST_DATABASE_URL`: semua paket `ok`, termasuk 9 test integrasi.
- `go test ./internal/server/` tanpa variabel: test integrasi ter-skip.
- `go vet ./...` dan `gofmt -l` bersih.
- Stack Docker di-rebuild; `npm run smoke:api` lulus, `otherCount=0`.

**Batasan:** test ini belum mencakup login Supabase (exit #1), voice (exit #5 via STT) karena belum ada provider, dan Playwright E2E di browser.
