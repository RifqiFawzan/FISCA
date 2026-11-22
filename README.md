# FISCA

AI-powered personal finance app: catat transaksi lewat suara atau chat, lihat ringkasan keuangan, dan kelola budget dalam satu tempat.

## Stack

| Layer | Teknologi |
|---|---|
| Web | Next.js (App Router) + TypeScript + Tailwind CSS v4 |
| API | Go + Gin |
| Database | PostgreSQL 17 (Docker Compose) |
| Migrations / queries | goose (sebagai library) + sqlc |
| Auth | Supabase JWT (RS256/JWKS) — mode `dev` untuk lokal |

Keputusan arsitektur lengkap ada di [`plan.md`](./plan.md), kebutuhan produk di [`PRD_FISCA.md`](./PRD_FISCA.md), dan sistem visual di [`DESIGN.md`](./DESIGN.md). Status pekerjaan ada di [`docs/backlog.md`](./docs/backlog.md), daftar issue lengkap di [`docs/issues.md`](./docs/issues.md), catatan verifikasi di [`docs/qa.md`](./docs/qa.md), dan eval parser di [`docs/eval/`](./docs/eval/) (`ai-cases.jsonl` 45 kasus + [`hasil.md`](./docs/eval/hasil.md)).

**Progres per 22 November 2026:** `ISSUE-01` s.d. `ISSUE-26` — **25 CLOSED (QA PASS)**, **0 OPEN**, **1 BLOCKED** (ISSUE-16, menunggu `OPENAI_API_KEY`). Nomor berikutnya: `ISSUE-27`.

## Status implementasi

Milestone M1–M7 selesai dan diverifikasi QA: fondasi Docker, ledger transaksi, idempotency, dashboard/analitik, pencarian riwayat bahasa alami, budget + proposal + approval, asisten (chat/kartu), transkripsi suara (proxy), rate limit, log terstruktur, dan seluruh halaman web yang terhubung ke API nyata.

Halaman: `/home`, `/history`, `/reports`, `/budget`, `/savings`, `/profile`, `/assistant`, dan `/login` (masuk/daftar email + kata sandi).

Yang **belum** ada (tidak diklaim selesai): verifikasi STT dengan provider hosted (ISSUE-16) dan pengujian CI otomatis. Tanpa `OPENAI_API_KEY`, asisten memakai heuristik deterministik Bahasa Indonesia dan endpoint STT mengembalikan `503 AI_UNAVAILABLE` — **kecuali** `OPENAI_STT_BASE_URL` menunjuk server STT lokal (service `stt`), yang memang tidak butuh key sama sekali.

Auth lokal email + kata sandi sudah berakhir (**ISSUE-15, CLOSED**), jadi jalur `dev:` tidak ada lagi.

## Prasyarat

- Docker Desktop + Docker Compose (jalur utama)
- Node.js 22+ dan npm (untuk lint/build lokal)
- Go 1.27+ (untuk test/build API lokal)

## Menjalankan dengan Docker (disarankan)

```bash
# 1. Siapkan environment (opsional; compose punya default)
cp .env.example .env

# 2. Build dan jalankan seluruh stack
npm run stack:up
```

Docker Compose menjalankan lima service:

| Service | Fungsi | Port host |
|---|---|---|
| `db` | PostgreSQL 17 + volume persisten | `5434` |
| `db-init` | `goose up` lalu seed development, selesai lalu exit | — |
| `api` | Go + Gin | `8080` |
| `web` | Next.js standalone | `3000` |
| `stt` | Speech-to-text lokal (speaches + Whisper, GPU CUDA) untuk jalur voice | `9000` |

Seed dilewati otomatis bila `APP_ENV=production`.

```bash
npm run stack:ps       # status + health
npm run stack:logs     # log api + web
npm run stack:down     # stop (volume tetap)
npm run stack:reset    # stop + hapus volume (data hilang)
```

- Web: http://localhost:3000
- API: http://localhost:8080
- Health: http://localhost:8080/health
- Postgres: `localhost:5434` (5432/5433 dihindari karena bisa dipakai instance lain)

Setelah mengubah kode, jalankan `npm run stack:up` lagi untuk rebuild image. Compose kini memakai `--env-file .env`, jadi `.env` di root adalah satu sumber konfigurasi (termasuk build arg web).

## Development tanpa Docker (hot reload)

Gunakan ini bila ingin hot reload; Postgres tetap dari Docker:

```bash
cp .env.example .env

# Dua instalasi terpisah: root tanpa "workspaces", jadi dependensi Next.js
# tidak ikut terpasang oleh npm install di root.
npm install                     # tooling root: playwright, typescript, openapi-typescript
npm --prefix apps/web install   # dependensi Next.js (lucide-react, dsb.)

npm run db:setup   # db:up + migrate + seed
npm run dev        # API + web bersamaan
```

Atau jalankan `npm run dev:api` dan `npm run dev:web` di terminal terpisah.

FISCA mengautentikasi dirinya sendiri (ISSUE-15): daftar atau masuk lewat `/login`
dengan email + kata sandi. API menerbitkan token sesi acak, menyimpan hanya
hash SHA-256-nya, dan mencabut token itu saat keluar.

```
# ambil token
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"anda@example.com","password":"kata-sandi-anda"}'
# lalu pakai pada endpoint terproteksi
Authorization: Bearer <token dari respons>
```

Tidak ada lagi token `dev:<uuid>` yang dipercaya apa adanya — mode itu dihapus
agar hanya ada satu jalur auth yang benar-benar teruji. Jalur `AUTH_MODE=jwt`
(verifikasi RS256 via JWKS) masih ada sebagai cadangan Supabase yang dorman;
terbukti oleh test di `internal/auth` dan `internal/server`, tapi tidak dipakai
dalam keadaan default.

### Variabel environment penting

| Variabel | Fungsi |
|---|---|
| `AUTH_MODE` | `session` (auth lokal, default) atau `jwt` (Supabase JWKS RS256, dorman) |
| `OPENAI_API_KEY` | Kosong → asisten memakai heuristik lokal; STT tetap jalan bila `OPENAI_STT_BASE_URL` diisi, kalau tidak → `503 AI_UNAVAILABLE` |
| `OPENAI_STT_BASE_URL` | Target transkripsi terpisah dari chat. Docker: `http://stt:8000/v1` (di-set compose); mode host: `http://localhost:9000/v1`. Kosong → memakai `OPENAI_BASE_URL` |
| `OPENAI_MODEL`, `OPENAI_STT_MODEL`, `OPENAI_BASE_URL`, `AI_TIMEOUT_SECONDS` | Konfigurasi gateway AI. `OPENAI_STT_MODEL` harus sama dengan model yang terpasang di service `stt` |
| `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST` | Token bucket per user; AI/STT otomatis setengahnya |
| `BUDGET_WARNING_THRESHOLD` | Ambang peringatan budget (default `0.8`) |
| `NEXT_PUBLIC_API_BASE_URL` | Di-bake saat build image web |

## Script

| Perintah | Fungsi |
|---|---|
| `npm run stack:up` / `stack:down` | Build + jalankan / hentikan seluruh stack Docker |
| `npm run stack:ps` / `stack:logs` | Status / log API + web |
| `npm run stack:reset` | Hapus container + volume database |
| `npm run dev` | API + web di host (butuh DB sudah jalan) |
| `npm run dev:api` / `npm run dev:web` | Jalankan salah satu di host |
| `npm run db:up` / `db:down` / `db:reset` | Kontrol container Postgres saja |
| `npm run db:migrate` / `db:rollback` / `db:status` | goose |
| `npm run db:seed` | Fixture development (menolak `APP_ENV=production`) |
| `npm run test:api` | `go test ./...` (termasuk test kontrak `docs/api/openapi.yaml`) |
| `npm run test:e2e` | Playwright E2E 9 alur wajib plan §12.5 + form masuk/daftar, viewport 1280 + 360 (butuh stack jalan) |
| `npm run gen:api` | Regenerasi `apps/web/lib/api.gen.ts` dari `docs/api/openapi.yaml` |
| `npm run lint:web` / `npm run build:web` | ESLint / production build |
| `npm run test:types` | `tsc -p tsconfig.test.json` — typecheck `tests/` |
| `npm run vet:api` | `go vet ./...` |
| `npm run auth:reset -- <email> [sandi]` | Reset password lokal + cabut semua sesi (ISSUE-24) |
| `powershell -File scripts/smoke.ps1` | Smoke end-to-end seluruh endpoint (butuh stack berjalan) |

`Taskfile.yml` menyediakan alias yang sama bila `task` terpasang.

## Testing

| Perintah | Cakupan |
|---|---|
| `npm run test:api` | Unit + integration Go (butuh `TEST_DATABASE_URL` bila ingin integration test; lihat `docs/qa.md`) |
| `npm run lint:web` / `npm run build:web` | ESLint + production build Next.js |
| `npm run test:e2e` | Playwright: 9 alur wajib plan §12.5 + form masuk/daftar (`auth-form.spec.ts`), dua viewport (1280 & 360) |
| `powershell -File scripts/smoke.ps1` | Smoke API end-to-end |

E2E berjalan terhadap stack Docker yang **sedang berjalan** (`npm run stack:up` dulu):

```bash
npm run stack:up
npm run test:e2e     # 50 test (25 definisi × 2 viewport), ±40 detik
npm run test:types   # typecheck seluruh tests/ (Playwright tidak memeriksa tipe)
```

Catatan E2E:
- Memakai Chrome sistem (`channel: "chrome"`), jadi tidak perlu unduh browser Playwright.
- Tiap test memakai **user baru** — rate limit API dihitung per user, dan berbagi satu user membuat test berikutnya kena `429`.
- Panggilan `register`/`login` berbeda: bucket-nya **per IP** dan sengaja ketat (`RATE_LIMIT_RPS/4`). Sebisa mungkin buat akun persiapan lewat API, bukan lewat form, agar test baru tidak menghabiskan kuota itu.
- `globalSetup` / `globalTeardown` menghapus semua user E2E (`email LIKE 'e2e-%@fisca.test'`); semua data mereka ikut terhapus (cascade), termasuk sesinya. Akun smoke sengaja tidak disentuh, karena penghapusannya mentok di `budget_categories.category_id` (`ON DELETE RESTRICT`).
- Alur voice memakai mikrofon palsu + STT yang di-mock: provider asli butuh `OPENAI_API_KEY` (ISSUE-16).

### Kontrak API

`docs/api/openapi.yaml` adalah sumber kebenaran bentuk respons. Dua hal dijaga darinya:

1. **Tipe TypeScript** — `npm run gen:api` menghasilkan `apps/web/lib/api.gen.ts`; tipe DTO di `lib/api.ts` hanya alias ke kontrak itu, jadi tidak ada bentuk wire yang ditulis dua kali.
2. **Test kontrak** — `go test ./internal/server/ -run Contract` memicu semua 28 endpoint dan memvalidasi setiap respons terhadap YAML. Mengubah DTO tanpa memperbarui kontrak membuat test ini gagal; begitu pula menghapus `card.type` atau kode error yang masih dipakai.

Setelah mengubah DTO Go, jalankan `npm run gen:api` dan test kontrak bersamaan.

## Struktur

```text
apps/
├── web/                 # Next.js + Dockerfile (standalone)
└── api/                 # Go + Gin + Dockerfile
    ├── cmd/api          # HTTP server
    ├── cmd/migrate      # goose runner (embedded SQL)
    ├── cmd/seed         # fixture runner
    ├── db/
    │   ├── migrations   # goose SQL
    │   ├── queries      # sqlc input
    │   └── seed
    └── internal/
        ├── ai/            # gateway OpenAI (intent JSON + transkripsi)
        ├── api/           # handler HTTP tipis
        ├── auth/ config/ httpx/ middleware/
        ├── infrastructure/{database,envfile}
        ├── period/        # resolusi periode half-open, tz pengguna
        ├── server/        # wiring router + middleware
        └── service/       # aturan bisnis (stateless, transaksional)
apps/web/
├── app/(app)/{home,history,reports,budget,profile}
├── app/(assistant)/assistant
├── components/{ui,finance,assistant,views,layout}
└── lib/{api,format,use-api}.ts
infra/
└── docker-compose.yml   # db + db-init + api + web + stt
docs/
├── backlog.md  issues.md  qa.md
```

## Docker image

- **API**: multi-stage `golang:1.27-alpine` → `alpine:3.22`, satu image berisi
  `api`, `migrate`, dan `seed`; berjalan sebagai user non-root.
- **Web**: multi-stage `node:22-alpine` dengan `output: "standalone"`, hanya
  membawa runtime yang dibutuhkan; berjalan sebagai user non-root.
- Healthcheck API memakai `/health` (termasuk ping DB); web mengecek `/home`.

## Catatan keamanan

- Tidak ada secret yang di-commit. Semua lewat `.env` lokal / secret store.
- Auth lokal (ISSUE-15): kata sandi di-hash **Argon2id** (64 MiB); token sesi
  disimpan ter-hash SHA-256 dan dicabut di server saat keluar. Tidak ada token
  yang dipercaya apa adanya — `AUTH_MODE` hanya `session` (default) atau `jwt`.
- Gagal masuk selalu menjawab "Email atau kata sandi salah" yang sama untuk
  email tak dikenal, akun tanpa kata sandi, dan kata sandi salah, sehingga login
  tidak bisa dipakai untuk menebak keberadaan akun.
- Endpoint auth memakai bucket rate limit sendiri; Argon2id menambah biaya per
  percobaan di atasnya.
- Command `seed` menolak berjalan saat `APP_ENV=production`.
- Saldo rekening diturunkan dari ledger transaksi, bukan kolom saldo yang bisa diubah langsung.
- Mutasi tulis wajib `Idempotency-Key`; retry tidak menggandakan transaksi.
- Perubahan budget harus lewat proposal yang disetujui; over-budget hanya peringatan.
- Rate limit per user (dan lebih ketat untuk AI/STT) mengembalikan `429 RATE_LIMITED`.
- Log operasional JSON memiliki `requestId` untuk korelasi; audit bisnis terpisah di tabel `audit_logs`.
