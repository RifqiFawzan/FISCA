# FISCA — Technical Plan (Phase 0)

**Dokumen:** `plan.md`
**Versi:** 0.1 — Draft untuk direview product owner
**Tanggal:** 8 November 2026
**Sumber acuan:** `PRD_FISCA.md` v1.2, `DESIGN.md` v1.0, `workflow.md`, dan screen Stitch "Fisca Application Project" (`2264071121943649738`)

> **Cara membaca dokumen ini.**
> `LOCKED` = sudah dikunci PRD/DESIGN, jangan diubah tanpa change log.
> `DECIDED` = keputusan teknis di dokumen ini, boleh dieksekusi.
> `NEED APPROVAL` = rekomendasi saya, **butuh satu kata dari Anda** sebelum Phase 1 menyentuh area tersebut.
> `OPEN` = sengaja ditunda, tidak boleh ditebak saat implementasi.

---

## 1. Tujuan Phase 0

Phase 0 belum menulis fitur. Output Phase 0:

1. Technical design yang cukup detail untuk memulai Phase 1 tanpa menebak.
2. Skema database (ERD + DDL-level rules) sebagai source of truth angka.
3. API contract awal (envelope, error, pagination, idempotency) agar frontend dan backend bisa paralel lewat mock.
4. Design token + komponen frontend yang diterjemahkan dari `DESIGN.md`.
5. Decision log: apa yang sudah tutup, apa yang masih butuh persetujuan.
6. Backlog awal per milestone + issue pertama yang siap dikerjakan.

Phase 0 **tidak** menghasilkan: kode aplikasi, UI hi-fi baru, atau finalisasi provider AI.

---

## 2. Ringkasan Stack dan Batasan

| Layer | Pilihan | Status | Catatan implementasi |
|---|---|---|---|
| Frontend | Next.js App Router + React + TypeScript | LOCKED | Route group `(app)` / `(auth)` |
| Styling | Tailwind CSS + CSS variables token | DECIDED | Token dari `DESIGN.md` §4 |
| Komponen UI | shadcn/ui sebagai basis, di-restyle ke token FISCA | DECIDED | Jangan pakai default theme shadcn |
| Ikon | `lucide-react` | DECIDED | Stitch memakai Material Symbols; lihat D-09 |
| Chart | Recharts | DECIDED | PRD §12.2, wajib label tanpa hover |
| Server state | TanStack Query | DECIDED | Invalidation setelah mutasi terkonfirmasi |
| Form + validasi | React Hook Form + Zod | DECIDED | Backend tetap validasi ulang |
| Backend | Go + Gin, modular monolith | LOCKED | Handler tipis, logic di service |
| Akses DB | `pgx/v5` + `sqlc` | DECIDED | Tanpa ORM |
| Migration | `goose` | DECIDED | SQL murni, reversible |
| Database | PostgreSQL 17 (Docker) | LOCKED | Volume persisten |
| Auth | **Auth lokal custom** (email+sandi, Argon2id, tabel `sessions`) — D-01 APPROVED | **APPROVED (10 Nov 2026)** | Kode Supabase dibiarkan dorman untuk kemungkinan SaaS |
| AI | OpenAI API + official Go SDK | **NEED APPROVAL** (D-02) | Model dipilih saat spike |
| STT | OpenAI Transcription | **NEED APPROVAL** (D-03) | Wajib benchmark Bahasa Indonesia |
| Testing | Go test + integrasi Postgres + Playwright | DECIDED | Lihat §12 |
| Repo | Monorepo (npm workspace + Go module terpisah) | DECIDED | Lihat §6 |

**Batasan keras yang berlaku di semua layer**

- Uang = integer rupiah (`BIGINT`). Tidak ada float di domain, API, maupun UI state.
- Saldo dan agregasi **hanya** dihitung backend/DB. Frontend tidak menjumlahkan apa pun untuk ditampilkan sebagai fakta.
- Transfer bukan income/expense.
- Status `pending`/draft tidak pernah masuk agregasi.
- AI tidak menerima SQL, tidak menetapkan `user_id`, tidak mengubah budget tanpa approval.

---

## 3. Open Decisions — rekomendasi final

| ID | Isu | Rekomendasi | Alasan | Status |
|---|---|---|---|---|
| O-01 | Account model | **Hybrid Accounts**: `cash` default otomatis saat onboarding, user bisa tambah `bank` / `ewallet`. Saldo **diturunkan** dari ledger + `opening_balance`, tidak ada kolom saldo mutable. | Mendukung transfer & purchase affordability tanpa risiko drift saldo | NEED APPROVAL |
| O-02 | Brand/name | **FISCA** | Sudah dipakai DESIGN.md v1.0 | CLOSED |
| O-03 | Visual system | Dark-first, Emerald, Plus Jakarta Sans, glass selective | Sudah dikunci DESIGN.md §4 | CLOSED |
| O-04 | Auth provider | **Custom auth Go lokal** (email+password, Argon2id, token sesi disimpan & di-hash server-side) | Pemakaian pribadi dulu; tidak bergantung layanan pihak ketiga. Kode Supabase dibiarkan dorman bila nanti jadi SaaS | **APPROVED (10 Nov 2026)** |
| O-05 | DB access | `pgx/v5` + `sqlc` + `goose` | Typed query, tanpa ORM, migrasi eksplisit | DECIDED |
| O-06 | AI provider/model | OpenAI; model dipilih lewat spike (§11.4), nama model di env | Hindari pinning model yang bisa deprecated | NEED APPROVAL |
| O-07 | STT provider | OpenAI Transcription; benchmark 40 perintah Indonesia (§12.4) | Perlu bukti akurasi nominal & tanggal | NEED APPROVAL |
| O-08 | Transcript retention | Transcript + audio metadata disimpan **30 hari**, audio mentah **tidak disimpan** di MVP | Data minimization | NEED APPROVAL |
| O-09 | Unusual spending baseline | Median + MAD per kategori; minimal **8 transaksi** dalam **90 hari**; flag bila `amount > median + 3×MAD` **dan** `amount ≥ Rp100.000` | Tahan outlier, menghindari false positive pada sampel kecil | NEED APPROVAL |
| O-10 | Budget period | Bulanan kalender, batas `[period_start, period_end)` pada timezone user | Sederhana dan cocok dengan UI | DECIDED |
| O-11 | Rollover | Tidak ada rollover otomatis | PRD §11.6 | DECIDED |
| O-12 | Warning threshold | 80% warning, 100% exceeded; disimpan sebagai konstanta config, bukan hardcode UI | Mudah diubah setelah pilot | DECIDED |
| O-13 | Refund accounting | Refund = transaksi income dengan `refund_of_id` → expense net = `expense - refund(linked)`. Tidak masuk MVP sampai aturan disetujui | Butuh definisi gross/net | OPEN |
| O-14 | Debt domain | Di luar MVP | PRD Non-Goals | CLOSED (out of scope) |
| O-15 | Currency | IDR saja, minor unit = 0, format `Rp1.250.000` | PRD §6.2 | DECIDED |
| O-16 | Conversation retention | 90 hari, ada delete per conversation | Cukup untuk debugging & privasi | NEED APPROVAL |
| O-17 | Savings goal model | Goal punya `account_id` tujuan + kontribusi tercatat sebagai transfer internal | Menghindari double counting | OPEN (Should Have) |
| D-01..D-12 | Keputusan teknis lain | Lihat §14 Decision Log | — | — |

---

## 4. Arsitektur Sistem

### 4.1 Bentuk

```text
Browser (mobile-first PWA-less responsive web)
        |
        | HTTPS, JSON, fetch/stream
        v
Next.js (App Router, RSC untuk shell, client untuk interaksi)
        |
        | Bearer JWT
        v
Go + Gin  ──────────────┐
  ├─ middleware: auth, request-id, rate-limit, recovery
  ├─ modules: auth/users/accounts/categories/transactions/
  │           budgets/analytics/assistant/savings
  └─ infrastructure: db (pgx+sqlc), ai (OpenAI), speech
        |                       |
        v                       v
   PostgreSQL             OpenAI API (chat + STT)
```

- **Modular monolith**: satu binary API, modul terpisah, tidak ada service mesh.
- Handler → Service → Repository. Gin handler tidak boleh berisi aturan bisnis maupun SQL.
- AI tool memanggil **service layer yang sama** dengan REST. Tidak ada jalur kedua yang melewati validasi.
- Redis/queue belum dipakai. Kalau muncul kebutuhan, ditambahkan di Phase 7.

### 4.2 Aturan dependensi

| Boleh bergantung pada | Tidak boleh |
|---|---|
| `modules/*/service` → `repository` + `shared` | `modules/*` → handler modul lain |
| `assistant` → service modul domain | `assistant` → SQL langsung |
| `analytics` → repository read-only | `analytics` → menulis data |
| `handler` → DTO + service | `handler` → `pgx` |

---

## 5. Data Model

### 5.1 Prinsip

1. Setiap tabel milik user punya `user_id UUID NOT NULL` (kecuali tabel referensi global).
2. Tidak ada `balance` mutable. Saldo = turunan ledger.
3. `transfer` adalah **satu baris** dengan `from_account_id` dan `to_account_id` → seimbang by construction.
4. Void = status, bukan delete.
5. Semua perubahan transaksi & budget menulis `audit_logs`.
6. Semua nominal `BIGINT` rupiah, `CHECK (amount > 0)`.

### 5.2 ERD

```text
users 1───n accounts
users 1───n categories
users 1───n transactions
users 1───n budgets 1───n budget_categories
users 1───n budget_proposals
budgets 1───n budget_proposals
users 1───n assistant_conversations 1───n assistant_messages
users 1───n drafts
users 1───n audit_logs
users 1───n idempotency_keys

categories 1───n transactions
accounts   1───n transactions (account_id | from_account_id | to_account_id)
transactions 1───n transactions (refund_of_id, future)
```

### 5.3 Tabel inti

**`users`**

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | UUID PK | = `auth.users.id` bila Supabase dipakai |
| `email` | CITEXT UNIQUE | not null |
| `display_name` | TEXT | not null |
| `timezone` | TEXT | default `Asia/Jakarta`, dipakai untuk periode budget & "hari ini" |
| `locale` | TEXT | default `id-ID` |
| `base_currency` | CHAR(3) | default `IDR` |
| `onboarded_at` | TIMESTAMPTZ | null = belum selesai onboarding |
| `created_at` / `updated_at` | TIMESTAMPTZ | — |

**`accounts`**

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID FK | |
| `name` | TEXT | unik per user (case-insensitive) di antara yang tidak diarsipkan |
| `type` | TEXT | `cash` / `bank` / `ewallet` / `other` |
| `opening_balance` | BIGINT | default 0; **bukan** saldo hasil perhitungan |
| `is_default` | BOOLEAN | tepat satu per user (`partial unique index`) |
| `archived_at` | TIMESTAMPTZ | null = aktif |
| `created_at` / `updated_at` | TIMESTAMPTZ | |

Saldo akun:

```sql
opening_balance
+ SUM(income.amount WHERE account_id = a.id AND status = 'active')
- SUM(expense.amount WHERE account_id = a.id AND status = 'active')
+ SUM(transfer.amount WHERE to_account_id = a.id AND status = 'active')
- SUM(transfer.amount WHERE from_account_id = a.id AND status = 'active')
```

**`categories`**

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID FK | default category di-seed per user saat onboarding |
| `name` | TEXT | unik per `(user_id, kind)` case-insensitive |
| `kind` | TEXT | `income` / `expense` |
| `icon` / `color` | TEXT | dipakai UI |
| `is_default` | BOOLEAN | true = seeded, boleh diarsipkan tapi tidak dihapus |
| `archived_at` | TIMESTAMPTZ | |

Default kategori mengikuti PRD §6.3. `Uncategorized` = `category_id NULL` (bukan baris kategori), agar laporan tidak bias.

**`transactions`**

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID FK | dari token, tidak pernah dari body |
| `type` | TEXT | `income` / `expense` / `transfer` |
| `amount` | BIGINT | `> 0` |
| `currency` | CHAR(3) | default `IDR`, `CHECK (currency = 'IDR')` di MVP |
| `description` | TEXT | not null, `btrim(length) > 0` |
| `category_id` | UUID FK NULL | wajib untuk income/expense **final**, null saat draft saja |
| `account_id` | UUID FK NULL | wajib income/expense |
| `from_account_id` | UUID FK NULL | wajib transfer |
| `to_account_id` | UUID FK NULL | wajib transfer, `<> from_account_id` |
| `payment_method` | TEXT NULL | `cash`/`qris`/`debit`/`transfer`/lain |
| `merchant_name` | TEXT NULL | |
| `occurred_at` | TIMESTAMPTZ | not null, default now, boleh lampau |
| `notes` | TEXT NULL | |
| `source` | TEXT | `manual` / `voice` / `chat` |
| `status` | TEXT | `active` / `voided` |
| `voided_at`, `void_reason` | TIMESTAMPTZ, TEXT | wajib bila voided |
| `refund_of_id` | UUID FK NULL | OPEN (O-13) |
| `created_at` / `updated_at` | TIMESTAMPTZ | |
| `version` | INT | optimistic lock untuk edit |

CHECK constraint per tipe:

```sql
(type = 'transfer' AND from_account_id IS NOT NULL AND to_account_id IS NOT NULL
   AND from_account_id <> to_account_id AND account_id IS NULL AND category_id IS NULL)
OR
(type IN ('income','expense') AND account_id IS NOT NULL
   AND from_account_id IS NULL AND to_account_id IS NULL)
```

Index:

- `(user_id, occurred_at DESC)` — list & cursor pagination
- `(user_id, status, occurred_at DESC)`
- `(user_id, category_id, occurred_at)` — budget & breakdown
- `(user_id, account_id, occurred_at)` + partial untuk transfer
- GIN `pg_trgm` pada `description` & `merchant_name` — natural search

**`budgets`** (1 aktif per user per periode)

| Kolom | Tipe |
|---|---|
| `id` | UUID PK |
| `user_id` | UUID FK |
| `period_start` | DATE |
| `period_end` | DATE (exclusive) |
| `overall_limit` | BIGINT |
| `currency` | CHAR(3) default IDR |
| `status` | `active` / `archived` |
| `version` | INT (dinaikkan setiap approval) |

`UNIQUE (user_id, period_start)`, `CHECK (overall_limit > 0)`, `CHECK (period_end > period_start)`.

**`budget_categories`**: `(id, budget_id, category_id, limit_amount BIGINT)` + `UNIQUE (budget_id, category_id)`.
Validasi `SUM(limit_amount) <= overall_limit` dilakukan di service **dan** dijaga trigger/constraint saat commit approval.

**`budget_proposals`**

| Kolom | Tipe |
|---|---|
| `id` | UUID PK |
| `user_id` | UUID FK |
| `budget_id` | UUID FK |
| `base_version` | INT — versi budget saat proposal dibuat |
| `status` | `pending` / `approved` / `rejected` / `expired` / `stale` |
| `changes` | JSONB — daftar operasi (`set_overall`, `set_category`, `move_allocation`) |
| `rationale` | TEXT |
| `impact` | JSONB — hasil kalkulasi before/after |
| `expires_at` | TIMESTAMPTZ (default +24 jam) |
| `decided_at` | TIMESTAMPTZ |

Approval sukses **hanya jika** `proposal.status = 'pending'` **dan** `budget.version = proposal.base_version`, dilakukan dalam satu transaksi DB (row lock `FOR UPDATE`), lalu `budget.version++`, `status='approved'`, tulis audit log.

**`assistant_conversations` / `assistant_messages`**

- conversation: `(id, user_id, title, created_at, last_message_at)`
- message: `(id, conversation_id, user_id, role user|assistant|tool, content TEXT, tool_calls JSONB, card JSONB, created_at)`
- `card` menyimpan payload kartu (transaction result, clarification, budget proposal, dst) supaya riwayat bisa dirender tanpa memanggil ulang AI.

**`drafts`** (menunggu klarifikasi)

`(id, user_id, conversation_id, payload JSONB, missing_fields TEXT[], status pending|resolved|expired, created_at, expires_at)`.
Draft **tidak** memengaruhi saldo, dashboard, laporan, atau budget. Auto-expire 24 jam.

**`audit_logs`**

`(id BIGSERIAL, user_id, actor user|ai|system, entity_type, entity_id, action, before JSONB, after JSONB, request_id, created_at)`.
Ditulis untuk: create/update/void transaction, budget create/update, approval/rejection proposal, perubahan account/category, delete conversation.

**`idempotency_keys`**

`(user_id, key, endpoint, request_hash, status_code, response_body JSONB, created_at)` + `UNIQUE (user_id, key, endpoint)`.
Request dengan key sama + hash sama → kembalikan response tersimpan (tanpa eksekusi ulang). Hash beda → `409 IDEMPOTENCY_MISMATCH`.

### 5.4 Dashboard & report query rules

- Satu repository `analytics` dengan query SQL eksplisit (sqlc) untuk: summary, cash flow per bucket, category breakdown, largest, monthly trend.
- Semua query difilter `status = 'active'` dan `user_id = $session_user`.
- Transfer dikecualikan dari income/expense, hanya muncul di cash-flow netral/badge.
- Periode dihitung dari `timezone` user: `occurred_at AT TIME ZONE $tz`.

---

## 6. Struktur Repo

```text
FISCA/
├── apps/
│   ├── web/                  # Next.js + Dockerfile (standalone)
│   └── api/                  # Go + Gin (go.mod sendiri) + Dockerfile
│       ├── cmd/              # api, migrate, seed
│       ├── db/
│       │   ├── migrations/   # goose, 0001_init.sql ...
│       │   ├── queries/      # sqlc input
│       │   └── seed/         # fixture development
│       └── internal/
│           └── infrastructure/database/
│               └── generated/  # sqlc output (committed)
├── docs/
│   ├── PRD_FISCA.md          # (dipindah dari root)
│   ├── DESIGN.md
│   ├── workflow.md
│   ├── plan.md
│   └── design-refs/          # (dihapus) screenshot + HTML Stitch; tarik ulang via scripts/pull-stitch-screens.cjs
├── infra/
│   ├── docker-compose.yml
│   └── postgres/
├── package.json              # root task runner (npm scripts)
├── Taskfile.yml              # task runner (dev, db, test)
├── .env.example
└── README.md
```

`apps/api` internal:

```text
cmd/api/main.go
internal/
├── config/            # env loading + validation
├── middleware/        # auth, request-id, ratelimit, recovery, cors
├── modules/
│   ├── users/ accounts/ categories/ transactions/
│   ├── budgets/ analytics/ assistant/ savings/
│   └── <modul>/       # handler.go service.go repository.go dto.go
├── infrastructure/
│   ├── database/      # pool, tx helper
│   ├── ai/            # client, tool registry, orchestrator
│   └── speech/        # transcription client
└── shared/            # money, period, apierror, pagination, validate
```

`apps/web` internal:

```text
app/
├── (auth)/sign-in|sign-up|forgot-password
├── (app)/home|history|reports|budget|profile
├── (assistant)/assistant           # layout hide bottom nav
└── api/health
components/  features/  hooks/  lib/  styles/
```

Pemetaan komponen wajib mengikuti `DESIGN.md` §9 (inventory), tanpa komponen satu-off.

---

## 7. API Contract (v1)

### 7.1 Konvensi

- Base path `/api/v1`, JSON camelCase, waktu RFC3339 UTC.
- Uang: `{"amount": 25000, "currency": "IDR"}`. Tidak ada string terformat dari backend.
- Auth: `Authorization: Bearer <jwt>`. Semua endpoint wajib auth kecuali health & auth callback.
- `X-Request-Id` diterima/di-generate, selalu dikembalikan.
- Sukses: `{ "data": ..., "meta": {...} }`
- Gagal: `{ "error": { "code": "...", "message": "...", "details": {...}, "requestId": "..." } }`
- Error code: `VALIDATION_ERROR` (422), `UNAUTHORIZED` (401), `FORBIDDEN` (403), `NOT_FOUND` (404), `CONFLICT` (409), `IDEMPOTENCY_MISMATCH` (409), `BUDGET_STALE` (409), `RATE_LIMITED` (429), `AI_UNAVAILABLE` (503), `INTERNAL` (500).
- Pagination cursor: `?limit=20&cursor=<opaque>` → `meta.nextCursor`.

### 7.2 Endpoint MVP

| Method | Path | Catatan |
|---|---|---|
| GET | `/me` | profil + preferensi + onboarding state |
| PATCH | `/me` | timezone, locale, display name |
| GET/POST | `/accounts` | POST body: name, type, openingBalance |
| PATCH | `/accounts/{id}` | rename/archive; tidak ada set balance |
| GET/POST | `/categories` | filter `kind` |
| PATCH | `/categories/{id}` | rename/archive |
| POST | `/transactions` | wajib `Idempotency-Key` |
| GET | `/transactions` | filter type/category/account/period/q, cursor |
| GET | `/transactions/{id}` | detail + audit trail ringkas |
| PATCH | `/transactions/{id}` | butuh `version` (optimistic lock) |
| POST | `/transactions/{id}/void` | wajib `Idempotency-Key` + reason |
| GET | `/transactions/search` | natural query → structured filter (dipakai assistant & UI) |
| GET | `/analytics/dashboard` | summary + cashflow + breakdown + recent + budget snapshot |
| GET | `/analytics/financial-summary` | period-based |
| GET | `/analytics/largest-transactions` | `limit`, filter period/type/category |
| GET | `/analytics/unusual-spending` | mengembalikan `sufficientData` + findings |
| GET | `/analytics/spending-comparison` | periode vs pembanding |
| GET | `/analytics/cash-flow` | bucket harian/mingguan/bulanan |
| GET/POST | `/budgets` | POST membuat budget + alokasi awal |
| GET | `/budgets/{id}` | detail |
| PATCH | `/budgets/{id}` | butuh `version` |
| GET | `/budgets/{id}/status` | actual, remaining, unallocated, per-category status |
| POST | `/budget-proposals` | body: budgetId, changes, rationale |
| GET | `/budget-proposals/{id}` | + impact |
| PATCH | `/budget-proposals/{id}` | edit draft |
| POST | `/budget-proposals/{id}/approve` | wajib `baseVersion` + `Idempotency-Key` |
| POST | `/budget-proposals/{id}/reject` | |
| POST | `/assistant/messages` | body: conversationId?, text, source; response: message + card + tool trace ringkas |
| GET | `/assistant/conversations` | list |
| GET | `/assistant/conversations/{id}` | messages |
| DELETE | `/assistant/conversations/{id}` | hapus riwayat |
| POST | `/speech/transcriptions` | multipart audio; response transcript + duration |
| GET | `/health` | liveness + db ping |

Endpoint tambahan yang saya sarankan dan **belum** ada di PRD §17: `/analytics/dashboard` (satu round-trip untuk Home), `GET /assistant/conversations`, `DELETE` conversation. Kalau disetujui, PRD §17 perlu ditambah.

**Ditambahkan saat implementasi Phase 1 (ISSUE-20, PRD §10.4/§10.6):** `GET`/`POST /savings-goals`, `GET`/`PATCH`/`DELETE /savings-goals/{id}`, dan `GET /purchase-advisor?price=`. Semuanya tercakup di `docs/api/openapi.yaml` dan contract test; PRD §17 belum menuliskannya.

### 7.3 Kontrak yang harus stabil untuk frontend paralel

- `GET /analytics/dashboard?period=2026-11` → bentuk final dipakai Home.
- `GET /budgets/{id}/status` → bentuk final dipakai Budget & kartu AI.
- `POST /assistant/messages` → `card.type` = `transaction_result | clarification | financial_summary | historical_search | largest_transaction | unusual_spending | budget_status | budget_proposal | recommendation | error`.

Bentuk `card` ini adalah **kontrak visual**: frontend hanya merender tipe yang dikenal, tidak mem-parsing teks bebas.

---

## 8. Aturan Finansial (implementasi wajib)

1. **Create transaction** (manual/voice/chat) melewati `TransactionService.Create` yang sama:
   validasi tipe → resolusi account (default bila tidak disebut, tapi **tidak dikarang** bila user punya >1 akun dan ambigu → clarification) → resolusi kategori (fallback `NULL` = Uncategorized) → simpan → tulis audit → invalidasi cache.
2. **Transfer tunggal**: satu baris, tidak ada jurnal ganda. Konsistensi dijamin constraint.
3. **Void** mengubah `status='voided'`; agregasi otomatis mengecualikan. Tidak ada hard delete.
4. **Edit** memakai `version`; mismatch → `409 CONFLICT` dengan data terbaru.
5. **Idempotency**: `Idempotency-Key` wajib pada create transaction, void, approve/reject proposal, dan dipakai `assistantTurnId` untuk capture dari AI.
6. **Clarification**: field wajib yang belum pasti → simpan `drafts`, balas `clarification` card. Tidak menyentuh `transactions`.
7. **Budget status** dihitung saat request; tidak ada budget counter yang disimpan.
8. **Approval** atomik + version check + audit (lihat §5.3).
9. **Over-budget**: tidak pernah menolak create transaction; response menyertakan `warnings[]` (`BUDGET_WARNING`, `BUDGET_EXCEEDED`, `budgetId`, `categoryId`, `exceededAmount`).
10. **Sukses** hanya dikirim setelah commit DB. Tidak ada "optimistic saved" dari AI.

---

## 9. AI Tool Gateway

### 9.1 Prinsip

- Tool schema ketat (JSON Schema), `additionalProperties: false`.
- `user_id` **tidak pernah** jadi parameter tool; diambil dari context.
- Semua tool memanggil service yang sama dengan REST.
- Orkestrasi: `intent → entity extraction → tool call → validasi → render card`.
- LLM tidak pernah menghasilkan angka final; angka diambil dari `tool_result` dan dirender backend.

### 9.2 Daftar tool (nama teknis)

| Tool | Jenis | Input inti | Guard |
|---|---|---|---|
| `create_transaction` | write | type, amount, description, occurredAt?, categoryId?, accountId?, merchant?, paymentMethod?, idempotencyKey | Validasi domain; ambiguous → `clarification` |
| `search_transactions` | read | q?, from?, to?, type?, categoryId?, accountId?, minAmount?, maxAmount?, limit | Scope user |
| `update_transaction` | write | transactionId, changes, expectedVersion | Butuh target unik |
| `void_transaction` | write | transactionId, reason | Butuh target unik |
| `get_financial_summary` | read | period | — |
| `get_dashboard` | read | period | — |
| `get_largest_transactions` | read | period?, type?, categoryId?, limit | — |
| `detect_unusual_spending` | read | period, categoryId? | Wajib kembalikan `sufficientData` |
| `get_budget_status` | read | budgetId?/period | — |
| `create_budget_proposal` | write (draft) | budgetId, changes, rationale | Tidak mengubah budget aktif |
| `update_budget_proposal` | write (draft) | proposalId, changes | Hanya status `pending` |
| `approve_budget_proposal` | write | proposalId, baseVersion | Atomik + audit |
| `reject_budget_proposal` | write | proposalId | — |
| `get_savings_analysis` | read | — | Fase lanjutan |

### 9.3 Perilaku pipeline

```text
transcript/text
  → intent classify (transaction_command | question | budget_change | correction | smalltalk)
  → bila transaction_command: extract → AccountResolver → CategoryResolver
      → lengkap?  ya  → create_transaction → transaction_result card
                  tidak → drafts + clarification card
  → bila question: pilih tool read → angka dari tool → render card (selalu sertakan periode)
  → bila budget_change: get_budget_status → create_budget_proposal (pending) → budget_proposal card
```

Aturan keras:

- Jangan tampilkan `transaction_result` sebelum `create_transaction` sukses.
- Pertanyaan tidak boleh masuk jalur create.
- Pertanyaan tanpa data → `historical_search` card dengan "tidak ditemukan".
- Data tidak cukup untuk unusual spending → nyatakan keterbatasan, bukan klaim.
- Setiap kartu analitik menyertakan `period`, `scope`, dan `limitations` bila relevan.

---

## 10. Frontend: Token dan Layout

### 10.1 Token → Tailwind

`app/styles/tokens.css` mendefinisikan CSS variable dari `DESIGN.md` §4, lalu `tailwind.config.ts` memetakan:

| Token DESIGN.md | CSS var | Tailwind |
|---|---|---|
| `background #0F131C` | `--bg` | `bg-background` |
| `canvas #0A0E16` | `--canvas` | `bg-canvas` |
| `surface-low #181C24` | `--surface-low` | `bg-surface-low` |
| `surface #1C2028` | `--surface` | `bg-surface` |
| `surface-high #262A33` | `--surface-high` | `bg-surface-high` |
| `surface-glass rgba(24,28,36,.70)` | `--surface-glass` | `bg-surface-glass backdrop-blur-[16px]` |
| `text-primary #F1F5F9` | `--text` | `text-foreground` |
| `text-secondary #94A3B8` | `--text-secondary` | `text-muted` |
| `primary #10B981` | `--primary` | `bg-primary text-primary` |
| `income #34D399` / `expense #FB7185` / `transfer #38BDF8` / `ai #06B6D4` | — | `text-income`, dst |
| `warning #FBBF24` / `danger #F43F5E` | — | `text-warning`, `text-danger` |

Aturan Tailwind:

- Angka finansial selalu `tabular-nums`.
- Glass hanya untuk panel/hero/floating, **bukan** semua card. Wajib punya fallback opaque (DESIGN.md §4.5).
- Glow hanya untuk primary action / AI processing / warning / danger, opacity sesuai DESIGN.md §4.6.
- `prefers-reduced-motion` mematikan waveform & transisi non-esensial.

### 10.2 Layout

| Viewport | Shell |
|---|---|
| `<768` | single column, bottom nav 5 item, assistant full-screen (nav hidden), safe-area padding |
| `768–1023` | adaptive 2 kolom, nav collapsible/rail |
| `≥1024` | sidebar persisten + content max-width, tabel di History, assistant pakai area konten |

Rute Assistant memakai layout sendiri yang menyembunyikan bottom nav; `Back` memakai `router.back()` dengan fallback `/home`.

### 10.3 Komponen prioritas (urutan build)

1. `AppShell` / `MobileBottomNavigation` / `AssistantActionButton` / `DesktopSidebar`
2. `BalanceCard` + `BalanceVisibilityToggle`, `IncomeExpenseSummary`, `NetCashFlowCard`
3. `TransactionListItem`, `TransactionList`, `TransactionDetailSheet`, `TransactionFilters`
4. `AssistantLayout`, `ConversationList`, `MessageBubble`, `ChatInput`, `VoiceRecorder`, `AudioWaveform`
5. `TransactionResultCard`, `ClarificationCard`, `FinancialInsightCard`, `HistoricalTransactionCard`, `BudgetProposalCard`
6. `OverallBudgetCard`, `CategoryBudgetItem`, `BudgetProgress`, `UnallocatedBudget`, `BudgetInsightCard`, `BudgetApprovalSheet`
7. `PeriodSelector`, `IncomeExpenseChart`, `CashFlowTrendChart`, `SpendingCategoryChart`
8. `EmptyState`, `LoadingState`, `ErrorState`, `Toast`, `ConfirmationDialog`

Komponen presentasional **tidak** boleh berisi perhitungan finansial atau format uang ad-hoc; semua lewat `lib/format.ts` + `lib/money.ts`.

---

## 11. Backend: Config, Docker, Environment

### 11.1 `infra/docker-compose.yml`

```yaml
services:
  db:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: fisca
      POSTGRES_PASSWORD: fisca
      POSTGRES_DB: fisca
    ports: ["5434:5432"]  # 5432/5433 bisa dipakai instance Postgres lain di host
    volumes: ["fisca_pgdata:/var/lib/postgresql/data"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fisca -d fisca"]
      interval: 5s
      timeout: 3s
      retries: 10
volumes:
  fisca_pgdata:
```

Selain `db`, Compose menjalankan `db-init`, `api`, dan `web`:

```yaml
services:
  db-init:            # image API, goose up + seed (seed dilewati di production)
    build: ../apps/api
    entrypoint: ["/app/migrate"]
    command: ["up"]
  api:                # Go + Gin, healthcheck /health
    build: ../apps/api
    ports: ["8080:8080"]
    depends_on: { db-init: { condition: service_completed_successfully } }
  web:                # Next.js standalone, healthcheck /home
    build: ../apps/web
    ports: ["3000:3000"]
    depends_on: { api: { condition: service_healthy } }
```

Kedua image bersifat multi-stage dan berjalan sebagai user non-root. Menjalankan
web/API di host tetap didukung untuk hot reload (`npm run dev`); Docker Compose
adalah jalur utama untuk menjalankan seluruh stack.

### 11.2 Environment (`.env.example`)

```env
# api
API_PORT=8080
DATABASE_URL=postgres://fisca:fisca@localhost:5434/fisca?sslmode=disable
AUTH_JWKS_URL=            # bila Supabase
AUTH_ISSUER=
AUTH_AUDIENCE=
OPENAI_API_KEY=
AI_MODEL=
STT_MODEL=
BUDGET_WARNING_THRESHOLD=0.8
RATE_LIMIT_RPS=10

# web
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_SUPABASE_URL=
NEXT_PUBLIC_SUPABASE_ANON_KEY=
```

Tidak ada secret di repo. Secret hanya di `.env` lokal/CI secret store.

### 11.3 Config validation

`internal/config` gagal start bila: `DATABASE_URL` kosong, `OPENAI_API_KEY` kosong saat fitur AI aktif, threshold di luar `(0,1]`.

### 11.4 Spike yang harus selesai sebelum Phase 4

1. Pilih model chat: bandingkan akurasi tool-calling & biaya pada 40 test case (§12.4).
2. Pilih model STT: bandingkan WER pada rekaman Indonesia informal.
3. Ukur latency end-to-end voice → kartu (target internal: < 6s p95 untuk perintah pendek).
4. Konfirmasi biaya per 1.000 perintah agar keputusan produk realistis.

---

## 12. Testing Strategy

### 12.1 Backend (Go)

- Unit: `shared/money`, period calculation, budget allocation/validation, unusual-spending math, transfer constraint.
- Integrasi (Postgres nyata, container): 
  - create/void/edit transaction + agregasi cocok,
  - transfer tidak mengubah total dana & tidak masuk income/expense,
  - idempotency: request ganda → satu transaksi,
  - over-budget → tersimpan + warning,
  - approval: sukses, reject, stale version, double approve,
  - cross-user access → 404/403 untuk setiap endpoint.
- Wajib: setiap bug finansial yang ditemukan QA menjadi test regresi.

### 12.2 Frontend

- Component test: bottom nav (5 item, assistant elevated), hide nav saat assistant, states assistant.
- Form validation, keyboard/safe-area behavior, reduced-motion.
- Playwright: flow §12.5.

### 12.3 Kontrak

- DTO Go ↔ tipe TypeScript dihasilkan/dicek dari OpenAPI (`openapi.yaml` di `docs/api/`), bukan ditulis dua kali manual.
- Contract test memastikan `card.type` dan shape dashboard tidak berubah tanpa update versi.

### 12.4 AI evaluation set (Bahasa Indonesia)

File: `docs/eval/ai-cases.jsonl`. Minimal 40 kasus, kategori:

| Kategori | Contoh | Ekspektasi |
|---|---|---|
| Nominal | "25 ribu", "1,2 juta", "2 juta 500 ribu", "150rb" | amount benar |
| Tanggal | "kemarin", "tgl 15 Oktober", "bulan lalu" | occurredAt benar (tz Asia/Jakarta) |
| Multi-transaksi | "beli makan 25 ribu dan kopi 18 ribu" | 2 transaksi |
| Transfer vs expense | "pindah 300rb dari BCA ke Cash" | transfer, bukan expense |
| Pertanyaan vs perintah | "berapa pengeluaran minggu ini?" | tidak membuat transaksi |
| Koreksi | "ubah kopi tadi jadi 15 ribu" | target benar / clarification |
| Ambiguitas | "bayar 50 ribu" | clarification, bukan tebakan |
| Analitik | largest, unusual spending | angka = query DB |
| Budget | "buat budget makanan 1 juta" | proposal pending, bukan langsung aktif |

Metrik: transcription accuracy, intent accuracy, entity accuracy, duplicate rate, clarification correctness, numerical answer accuracy. Ambang go/no-go ditetapkan setelah run pertama.

### 12.5 E2E wajib (dari PRD §22.4)

1. Register → onboarding (cash account + default kategori) → expense manual → dashboard berubah.
2. Voice expense → STT → parse → validation → tersimpan.
3. Input ambigu → draft → clarification → tersimpan.
4. Retry ganda → satu transaksi.
5. Largest transaction = query DB.
6. Unusual spending dengan data kurang → menyatakan keterbatasan.
7. Proposal → approve → budget berubah + audit log.
8. Over-budget → tersimpan + warning.
9. User A akses data User B → ditolak.
10. Assistant mobile → nav hilang → Back kembali.

---

## 13. Milestones dan Issue Awal

| Milestone | Isi | Exit criteria |
|---|---|---|
| **M0** (dokumen ini) | plan.md + OpenAPI draft + token mapping | Anda approve D-01..D-12 |
| **M1 Foundation** | monorepo, docker compose db, goose, sqlc, Gin skeleton, health, config, auth middleware, CI lint/test | `task dev` jalan, `/health` 200, migrasi 0001 sukses |
| **M2 Financial Core** | accounts, categories, transactions (CRUD+void+idempotency), history list/search, audit log | E2E #1, #4 lulus; cross-user test lulus |
| **M3 Dashboard & Budget** | analytics queries, dashboard, reports, combined budget, proposal + approval, warning | E2E #7, #8 lulus |
| **M4 AI Capture** | assistant endpoint, conversation, STT, intent/entity, hybrid recording, cards | E2E #2, #3 lulus; eval ≥ baseline |
| **M5 Intelligence** | historical search natural, largest, unusual spending, analyst cards | E2E #5, #6 lulus |
| **M6 Advisor** | budget suggestion, conversational budget end-to-end, savings/purchase planning | proposal flow via chat lulus |
| **M7 Pilot** | security review, rate limit, observability, performance, deploy | MVP exit criteria PRD §23 (20 item) |

### Issue siap-kerja pertama (M1)

**FISCA-001 — Bootstrap monorepo & dev environment**
Acceptance criteria:
- `apps/web` (Next.js + TS + Tailwind + token FISCA) dan `apps/api` (Go + Gin) ada dan bisa start.
- `infra/docker-compose.yml` menyalakan Postgres 17 dengan volume persisten + healthcheck.
- `Taskfile.yml`: `task dev`, `task db:up`, `task db:migrate`, `task test`.
- `.env.example` lengkap, `.gitignore` menutup `.env`, `node_modules`, binary Go.
- `/health` mengembalikan `{status:"ok", db:"ok"}`.

**FISCA-002 — Migration 0001 + sqlc setup**
AC: tabel `users/accounts/categories/transactions/audit_logs/idempotency_keys` dibuat; constraint & index §5 terbentuk; `sqlc generate` bersih; `goose down` reversible.

**FISCA-003 — Auth middleware + user scope**
AC: request tanpa/invalid token → 401 envelope standar; token valid → context berisi `userID`; test cross-user gagal akses.

**FISCA-004 — Design token & app shell**
AC: token DESIGN.md §4 tersedia sebagai CSS var + Tailwind; bottom nav 5 item dengan Assistant elevated; desktop sidebar; assistant route menyembunyikan nav; tidak ada overflow horizontal 360px.

---

## 14. Decision Log

| ID | Keputusan | Status | Dampak bila diubah |
|---|---|---|---|
| D-01 | Auth: **auth lokal custom** (email + kata sandi, Argon2id, tabel `sessions`) — pengganti Supabase | **APPROVED (10 Nov 2026)** | Dipilih user: FISCA untuk pribadi dulu, SaaS hanya mungkin nanti (ISSUE-15). Termasuk **penghapusan `AUTH_MODE=dev`** demi satu jalur auth. Cabang Supabase dipertahankan sebagai kode dorman; `plan.md` sebelumnya menandai ini NEED APPROVAL. Reset-password flow **belum** dikerjakan — lihat backlog |
| D-02 | AI: OpenAI SDK, model via env | NEED APPROVAL | Provider lain → adaptor di `infrastructure/ai` |
| D-03 | STT: OpenAI Transcription | NEED APPROVAL | Butuh benchmark; fallback chat selalu tersedia |
| D-04 | Saldo diturunkan dari ledger, tanpa kolom `balance` | DECIDED | Bila butuh performa: tambah materialized view / cache, bukan kolom mutable |
| D-05 | Transfer = 1 baris (from/to) | DECIDED | Model 2-leg akan butuh jurnal + rekonsiliasi |
| D-06 | `Uncategorized` = `category_id NULL` | DECIDED | Baris kategori palsu akan mencemari laporan |
| D-07 | Idempotency via tabel + `Idempotency-Key` | DECIDED | — |
| D-08 | Budget = periode bulanan, satu aktif per periode | DECIDED | Periode kustom butuh perubahan unique constraint |
| D-09 | Ikon UI: `lucide-react` (bukan Material Symbols) | DECIDED | Mudah diganti; hindari dua ikon set |
| D-10 | Frontend/Backend paralel lewat OpenAPI + mock | DECIDED | — |
| D-11 | Aggregasi angka hanya lewat query backend | DECIDED | Dilarang oleh PRD |
| D-12 | Assistant `card.type` sebagai kontrak visual | DECIDED | Perubahan butuh versi kontrak |
| D-13 | Full stack via Docker Compose (db-init + api + web), host dev tetap didukung | DECIDED | Tanpa Docker: jalankan `npm run dev` + Postgres lokal |

---

## 15. Risiko dan Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Akurasi STT Bahasa Indonesia informal rendah | Voice-first gagal, user pindah ke chat | Benchmark 40 kasus sebelum Phase 4; tampilkan transkrip sebelum eksekusi bila confidence rendah |
| LLM mengarang angka | Kepercayaan hancur | Angka hanya dari tool result; render di backend; eval numerik wajib |
| Ambiguitas account pada user multi-akun | Salah catat | Default hanya bila 1 akun; selain itu clarification |
| Drift saldo | Laporan salah | Saldo turunan ledger + test agregasi |
| Stale budget proposal | Perubahan tidak sah | `base_version` + row lock + `409 BUDGET_STALE` |
| Duplikasi transaksi akibat retry | Data ganda | Idempotency key wajib + test |
| Cakupan desain tidak lengkap (Profile, Transaction Detail, desktop) | Implementasi menebak | Desain halaman tersebut dibuat saat milestone terkait, mengikuti token; tidak menebak perilaku |
| Stitch screen memuat konten carbon/ESG | Melanggar DESIGN.md §3.3 | Semua konten carbon dibuang; tidak ada metrik emisi di UI |
| Bug `stitch-pro-mcp@0.2.0` (fetch tanpa API key) | Tidak bisa tarik screen via MCP | Sudah di-workaround via REST API; jangan andalkan `sp_screens` |
| **Akar masalah `sp_screens` dipastikan** — `fetchContent(url)` memanggil `fetch(url)` **tanpa header sama sekali** (`dist/index.js:151`, salinan di `dist/bin/cli.js:151`) | Unduhan selalu 401 lalu dibungkus jadi `Failed to fetch: <url>` | Teruji 22 Nov 2026 (`scripts/probe-stitch-mcp.cjs`, `probe-stitch-auth.cjs`): URL yang diminta MCP ternyata **valid** (itu `htmlCode.downloadUrl`, bukan URL salah) → **401 tanpa kunci, 200 dengan `X-Goog-Api-Key` maupun `?key=`**; `apikey` header dan `Bearer` justru ditolak. Perbaikan satu baris: sertakan `x-goog-api-key` di `fetchContent`. **Sengaja tidak dipatch** — paketnya ada di npm cache sehingga hilang saat cache dibersihkan dan tidak bisa direproduksi. Pelaporan ke `github.com/LuciferDono/stitch-pro-mcp/issues`. Sementara tetap REST |
| `sp_projects` mengembalikan `screenCount: 0` untuk semua project | Jumlah screen tidak terbaca dari MCP | Defek terpisah dari bug di atas; daftar project tetap terbaca, hanya angkanya nol. REST mencatat 23 screen untuk project FISCA |

---

## 16. Yang Dibutuhkan dari Anda Sekarang

1. **Approve/ubah** item `NEED APPROVAL`: O-01, O-04, O-06, O-07, O-08, O-09, O-16 (+ D-01..D-03).
2. Konfirmasi 3 endpoint tambahan di §7.2 (`/analytics/dashboard`, list & delete conversation) supaya PRD §17 ikut diperbarui.
3. Konfirmasi apakah `PRD_FISCA.md`, `DESIGN.md`, `workflow.md` dipindah ke `docs/` atau tetap di root.
4. Setelah itu saya lanjut ke FISCA-001 (bootstrap monorepo) — atau mulai dari OpenAPI + migration dulu bila Anda mau backend-first.

---

## 17. Status Implementasi (Phase 1, 8 November 2026)

Dokumen di atas tetap menjadi rancangan Phase 0. Bagian ini mencatat apa yang
benar-benar terbangun dan **deviasi** dari keputusan `DECIDED`, agar tidak ada
klaim yang tidak sesuai kode.

### 17.1 Yang terbangun

- M1–M7 selesai: fondasi Docker, ledger, idempotency, analitik, pencarian
  natural, budget + proposal + approval, asisten chat/kartu + STT proxy,
  hardening (rate limit + log JSON), dan 6 halaman web terhubung API.
- Kontrak `card.type` §7.3 dipatuhi; `AssistantCardView` hanya merender tipe
  yang dikenal.
- Aturan finansial D-07/D-11/D-12 dijaga: agregasi backend, uang integer rupiah,
  over-budget warning-only, AI tidak pernah menulis `user_id`.
- Backlog, issue dengan acceptance criteria, dan catatan QA ada di `docs/`.

### 17.2 Deviasi dari tabel §2 (DECIDED, bukan LOCKED)

| Rencana | Implementasi | Alasan |
|---|---|---|
| TanStack Query | `useApi` + `reload()` eksplisit | Menghindari dependensi baru; invalidasi terjadi hanya setelah mutasi dikonfirmasi server |
| Recharts | Bar chart CSS + label sumbu/legenda/skala | Bundle lebih kecil, tidak ada dependensi chart; nilai per-batang masih memakai `title` (belum memenuhi "label tanpa hover" untuk setiap titik) |
| React Hook Form + Zod | Form controlled + `parseAmountInput` | Backend tetap sumber kebenaran validasi; parser rupiah informal diuji unit test |
| shadcn/ui | Komponen sendiri di `components/ui/*` | Token `DESIGN.md` dipakai langsung; tidak ada default theme pihak ketiga |
| OpenAPI → DTO TypeScript | Tipe di-alias dari `apps/web/lib/api.gen.ts` yang di-generate dari `docs/api/openapi.yaml` | `npm run gen:api`; contract test menangkap drift |
| `text-muted-dim` `#64748B` (DESIGN.md §4) | `#8493AB` | `#64748B` hanya 3,0–4,1:1 di permukaan gelap dan gagal WCAG AA (4,5:1); `#8493AB` lulus di semua permukaan (4,61 terburuk) dan tetap membedakan hierarki dari `muted`. Deviasi warna ini, bukan peran |
| `danger` `#F43F5E` (DESIGN.md §4) | `#F66981` | `#F43F5E` hanya 3,9–4,4:1 di atas chip/alert yang memakai tint-nya sendiri (badge 4,16); `#F66981` lulus di semua latar nyata (4,60 terburuk) dan tetap merah yang sama |
| `primary` `#10B981` (DESIGN.md §4) | `#4EDEA3` (+ hover `#46C893`, deep `#3EB282`) | Mengikuti aset design system Stitch "Dark Carbon Glass" (primary-fixed-dim / surface-tint). Lulus kontras lebih baik daripada nilai lama: teks 7,20 vs 4,84; tombol 7,73 vs 5,20. **Bonus:** hover & deep lama GAGAL (3,50 / 2,41) — Lighthouse tidak menangkap karena hanya mengaudit keadaan diam; kini 6,25 / 4,96. Glow & `::selection` ikut disamakan |
| `ai` `#06B6D4` / `analytics` `#6366F1` (DESIGN.md §4) | `#4CD7F6` (secondary Stitch) / `#C0C1FF` (tertiary Stitch) | Mengikuti palet Dark Carbon Glass; keduanya lulus 7,23 dan 7,20 di permukaan terberat |
| Teks `text-primary`/`text-secondary`/`text-muted` + `danger` (hasil audit ISSUE-21/23: `#F1F5F9`/`#94A3B8`/`#8493AB`/`#F66981`) | `#DFE2EE` / `#BBCABF` / `#86948A` / `#FFB4AB` | Arah product owner via `Color_Design.md` (blok `colors:` = token Stitch "Luminescent Carbon Fintech"); permukaan sudah sama, hanya teks & danger yang berbeda. Lulus AA di semua permukaan FISCA — terburuk `text-secondary` 7,20, `text-muted` 5,14, `danger` 8,12, `text-primary` 9,51 — diaudit `tests/audit-color-yaml.mjs`. Masuk `DESIGN.md` §4.1 + change log §1.3 pada22 Nov 2026 |
| shadcn/ui (utuh) | **Satu** komponen registry: `message-scroller` (auto-scroll chat asisten) + `components.json` (preset base-nova) & `lib/utils.ts`; dependensi baru: `@shadcn/react`, `@base-ui/react`, `class-variance-authority`, `cn`, `tw-animate-css` | Permintaan product owner (23 Nov). Init shadcn sempat menimpa `button.tsx`, `layout.tsx` (font Geist), dan `globals.css` (tema terang di `:root`) — **semua dibatalkan**; yang dipertahankan hanya import utilitas `tw-animate-css` + `shadcn/tailwind.css` (tanpa warna/font). Button FISCA tetap dipakai komponen baru (default-nya diselaraskan `size="sm"`). Verifikasi: `tsc`/`eslint`0 · `tests/verify-scroller.mjs` PASS · e2e **50/50** |

> **Pembaruan 12 Nov 2026:** kelima deviasi warna di atas **kini sudah masuk ke `DESIGN.md` §4.1**, disertai change log di `DESIGN.md` §1.3 — sesuai aturan bahwa dokumen `LOCKED` tidak boleh diubah tanpa change log. Baris di atas karena itu menjadi catatan sejarah: kode dan DESIGN.md tidak lagi berselisih. Pengerjaannya di `docs/issues/ISSUE-23.md`, keputusan revisinya di sesi 12 Nov 2026.

> **Pembaruan 22 Nov 2026:** empat nilai teks/danger pada baris terakhir tabel di atas (arah `Color_Design.md`) **kini juga sudah masuk ke `DESIGN.md` §4.1 + change log §1.3** — kembali selaras, tanpa deviasi tersisa untuk warna ini.

Deviasi ini **tidak** mengubah hal yang `LOCKED` (Next.js App Router, Go + Gin,
PostgreSQL, Docker Compose) maupun keputusan finansial. Bila product owner
menghendaki library di atas, perubahannya bersifat substitusi lokal dan tidak
menyentuh kontrak API.

### 17.3 Belum diimplementasi

Savings/purchase planning, login UI Supabase, evaluasi AI 40 kasus
(`docs/eval/ai-cases.jsonl`), Playwright component test, parsing multi-transaksi
satu kalimat, koreksi "ubah kopi tadi jadi 15rb", dan tanggal relatif "kemarin".
Mode `jwt` tersedia di backend namun belum diuji terhadap provider sungguhan.
Rate limit bersifat in-process per replica.

---

**End of plan.md — FISCA Phase 0 v0.1 + status implementasi Phase 1**
