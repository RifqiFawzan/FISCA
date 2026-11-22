# ISSUE-03 — Transaksi & idempotency (M3)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. `POST /transactions`, `POST /transactions/{id}/void`, `POST /budget-proposals/{id}/approve` wajib `Idempotency-Key`.
2. Request ulang dengan key sama mengembalikan respons tersimpan yang identik, tanpa transaksi baru.
3. Key sama dengan body berbeda → `409 IDEMPOTENCY_MISMATCH`.
4. Key tanpa header → `422 VALIDATION_ERROR`.
5. `PATCH /transactions/{id}` butuh `version`; `void` mengubah status, bukan menghapus baris.

**Implementasi:** reservasi `INSERT ... ON CONFLICT DO NOTHING` di luar transaksi kerja, `CompleteIdempotencyKey` di dalam transaksi kerja, `AbortIdempotency` saat gagal.

**QA:** smoke (`first=201`, `replay=201 sameId=True`), `curl.exe` raw body menghasilkan `IDEMPOTENCY_MISMATCH` dan `VALIDATION_ERROR`, `PATCH` → `version=2`, `void` → `status=voided`. → **PASS**
