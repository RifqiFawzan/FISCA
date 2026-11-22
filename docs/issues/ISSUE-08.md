# ISSUE-08 — Klarifikasi draft (E2E #3)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. Pesan ambigu membuat draft `pending` dengan kartu `clarification` dan `missingFields`.
2. Balasan berikutnya melengkapi draft, menyimpan transaksi, dan menandai draft `resolved`.
3. Pesan yang jelas merupakan pertanyaan/perintah baru tidak tersedot sebagai jawaban draft.
4. Draft tidak pernah masuk agregasi saldo/laporan/budget.

**Implementasi:** `GetPendingDraft` / `UpdateDraft` / `ResolveDraft`, `tryResolveDraft` di `internal/service/assistant.go`.

**QA:** `"catat 25rb"` → `clarification["description"]`; `"kopi susu"` → `transaction_result` dengan `toolTrace=["intent","draft_resolved"]`; `"berapa pengeluaran bulan ini"` → `financial_summary`. → **PASS**
