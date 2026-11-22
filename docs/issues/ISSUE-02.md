# ISSUE-02 — Ledger & data master (M2)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. Semua nominal tersimpan sebagai integer rupiah (`BIGINT`), bukan float/string.
2. Saldo akun dihitung dari ledger (`opening_balance` + income/expense/transfer aktif); tidak ada endpoint set saldo.
3. User baru otomatis mendapat akun `Cash` dan kategori default.
4. Setiap endpoint memfilter `user_id`; data user A tidak muncul untuk user B.
5. Setiap mutasi menulis `audit_logs`.

**Implementasi:** `0001_init.sql`, `internal/service/{users,accounts,categories,transactions}.go`, `internal/api/`.

**QA:** `go test ./...`, smoke `/me`, `/accounts`, `/categories`, detail transaksi (`audit` berisi 1 baris), dan percobaan akses lintas user → `otherCount=0`. → **PASS**
