# ISSUE-06 — Budget & advisor (M6)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. Perubahan budget lewat `POST /budget-proposals` → status `pending`; baru berlaku setelah `approve`.
2. `approve` memakai `baseVersion` + `Idempotency-Key`, menolak versi basi (`BUDGET_STALE`).
3. Transaksi yang melebihi budget tetap tersimpan dengan `BUDGET_WARNING` (never reject).
4. Asisten tidak pernah menetapkan `user_id`; semua write memakai identitas dari token.
5. Kartu asisten sesuai kontrak `card.type` (`plan.md` §7.3).

**Implementasi:** `internal/service/{budgets,assistant}.go`, `internal/api/{budgets,assistant}.go`, `components/assistant/assistant-card.tsx`.

**QA:** smoke (`proposal=201` → `approve=200 limit=1500000 version=5`), transaksi 420000/500000 → `warnings[0].type=BUDGET_WARNING`, E2E browser: "ubah budget jadi 2jt" → kartu `budget_proposal` → klik Setujui → budget versi 4/5. → **PASS**
