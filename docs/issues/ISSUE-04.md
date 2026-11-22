# ISSUE-04 — Dashboard & analitik (M4)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. Semua agregasi (summary, breakdown, cash-flow, largest, unusual) dihitung SQL/backend, bukan frontend.
2. Periode half-open `[start, end)` dan memakai timezone pengguna.
3. `unusual-spending` selalu mengembalikan `sufficientData` + keterbatasan saat data kurang.
4. Tidak ada UI yang menampilkan "tersimpan" sebelum respons server diterima.

**Implementasi:** `internal/period`, `internal/service/analytics.go`, `db/queries/analytics.sql`, `components/views/home-view.tsx`, `reports-view.tsx`.

**QA:** `/analytics/dashboard?period=this_month` → `income/expense/net` konsisten dengan daftar transaksi; unit test `internal/period` menguji batas bulan, `7d/30d/90d/1y`, `YYYY-MM`, `YYYY-MM-DD`, dan penolakan input invalid; form create baru menampilkan notifikasi setelah `await` API. → **PASS**
