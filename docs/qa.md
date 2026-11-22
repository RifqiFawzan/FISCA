# FISCA — Catatan Verifikasi QA

Tanggal: 8 November 2026. Lingkungan: Windows, Docker Desktop, PostgreSQL 17 di port 5434, API `:8080`, web `:3000`, API host uji `:8081`.

## 1. Build & static checks

| Perintah | Hasil |
|---|---|
| `cd apps/api && go build ./...` | exit 0 |
| `cd apps/api && go vet ./...` | exit 0 |
| `cd apps/api && go test ./...` | semua paket `ok` (termasuk `period`, `service`, `middleware`, `config`, `auth`, `database`) |
| `& $USERPROFILE\go\bin\sqlc.exe generate` | exit 0 (generated code selalu diregenerasi setelah edit SQL) |
| `cd apps/web && npx tsc --noEmit` | exit 0 |
| `cd apps/web && npm run lint` | exit 0, 0 warning |
| `cd apps/web && npm run build` | 10 route statis, compiled successfully |
| `docker compose --env-file .env -f infra/docker-compose.yml config` | `NEXT_PUBLIC_API_BASE_URL`, `RATE_LIMIT_BURST`, `OPENAI_MODEL`, `AI_TIMEOUT_SECONDS` terisi |
| `npm run stack:up` | `fisca-api:dev` + `fisca-web:dev` built; `db`, `db-init`, `api`, `web` healthy |

## 2. Unit test yang ditambahkan

- `internal/period/period_test.go`: batas bulan ini/bulan lalu, jendela 7/30/90 hari, `YYYY-MM`, `YYYY-MM-DD`, penolakan input garbage, sifat half-open.
- `internal/service/search_test.go`: parser rupiah, deteksi periode, pembersihan keyword sisa, `cleanDescription`, heuristic intent, pencocokan kategori.
- `internal/middleware/ratelimit_test.go`: burst token bucket, isolasi per identitas, envelope `429 RATE_LIMITED`, matcher rute AI/STT.

## 3. Smoke API end-to-end

Script: [`scripts/smoke.ps1`](../scripts/smoke.ps1) (`powershell -ExecutionPolicy Bypass -File scripts/smoke.ps1`, dapat diarahkan lewat `FISCA_BASE`). Dijalankan terhadap `:8080` (Docker) dan `:8081` (host).

| Skenario | Hasil |
|---|---|
| `GET /me` | `200`, `Dev User`, `Asia/Jakarta` |
| `GET /accounts` / `/categories` | saldo ledger-derived, 9 kategori default |
| POST transaksi + replay key sama | `201` + `201`, `sameId=True` |
| Key sama, body beda | `409 IDEMPOTENCY_MISMATCH` (diverifikasi raw JSON via `curl.exe`) |
| Tanpa `Idempotency-Key` | `422 VALIDATION_ERROR` |
| `GET /transactions`, detail, `PATCH` | `count>0`, `auditRows=1`, `version=2` |
| `/analytics/dashboard` | `income/expense/topCategories` konsisten |
| `/transactions/search?q=pengeluaran bulan ini di atas 10rb` | `period=this_month, type=expense, minAmount=10000, text=""`, hasil > 0 |
| Budget + transaksi 420000 | `201` + `BUDGET_WARNING` |
| Proposal + approve | `201` → `200`, `version` naik |
| Asisten heuristic: create / summary / search | `transaction_result`, `financial_summary`, `historical_search` |
| Konversasi list + history | `roles=user,assistant` |
| `/speech/transcriptions` tanpa file | `422 VALIDATION_ERROR` |
| `/speech/transcriptions` dengan file dummy, tanpa API key | `503 AI_UNAVAILABLE` |
| Isolasi user lain (`dev:2222…`) | `otherCount=0` |
| `void` | `status=voided` |
| Klarifikasi draft | `clarification` → balasan → `transaction_result` (`draft_resolved`) → pertanyaan baru tidak tersedot |

## 4. Rate limit & observability

- 30 request `GET /me` berturut-turut → 24× `200`, 6× `429` dengan header `Retry-After: 1` dan body `RATE_LIMITED`.
- `docker logs fisca-api` menghasilkan satu baris JSON per request: `{"time":...,"level":"WARN","msg":"http_request","requestId":...,"route":"/api/v1/me","status":429,"latencyMs":0,"bytes":138,"userId":...}`.
- Log 5xx memakai level `ERROR`, 4xx `WARN`, 2xx `INFO`. Audit bisnis tetap di tabel `audit_logs`.

## 5. Verifikasi browser

- Desktop (Review pane, ~1280px): `/home` menampilkan saldo, mini-stat, budget, kategori, transaksi terakhir; tanpa console error.
- Mobile 360px (CDP `Emulation.setDeviceMetricsOverride`): `scrollWidth=360` untuk `/home`, `/history`, `/reports`, `/budget`, `/profile`, `/assistant`; tidak ada elemen yang keluar viewport (kecuali elemen dekoratif blur di dalam kartu `overflow-hidden`).
- Screenshot 360px melalui `Page.captureScreenshot`: bottom nav 5 item, chip periode muat, asisten workspace tanpa bottom nav.
- Bug yang ditemukan & diperbaiki: `/profile` sempat `RangeError: Invalid time value` karena `GET /accounts` mengembalikan `createdAt=""`. Root cause: `ListAccountBalances` tidak men-select `a.created_at`. Perbaikan: query + `GROUP BY` diperbarui, sqlc diregenerate, DTO diisi, dan formatter frontend dibuat toleran terhadap tanggal kosong.

## 6. E2E interaksi (klik nyata di browser)

| Alur | Hasil |
|---|---|
| Home → "Catat transaksi" → nominal `17rb`, deskripsi "Uji browser es teh" → Simpan | notifikasi "berhasil dicatat" muncul setelah respons server; saldo `-Rp 166.000` → `-Rp 183.000`, transaksi muncul di daftar |
| Asisten → klik saran "berapa pengeluaran bulan ini" | kartu `financial_summary` dengan angka `Rp 183.000` |
| Asisten → "ubah budget jadi 2jt" | kartu `budget_proposal` + pesan "Perubahan baru berlaku setelah kamu setujui" |
| Budget → klik "Setujui" | "Proposal disetujui", limit `Rp 2.000.000`, versi naik |
| Riwayat → cari "pengeluaran bulan ini di atas 10rb" | banner `Filter: bulan ini, ≥ Rp10.000, pengeluaran` + daftar hasil |
| Console error pada semua alur di atas | 0 |

## 7. Yang belum diverifikasi / risiko

- Tidak ada `OPENAI_API_KEY`, jadi jalur AI + STT sungguhan (provider OpenAI) belum diuji; hanya fallback heuristik dan `503 AI_UNAVAILABLE` yang diuji.
- Tidak ada Supabase project, jadi mode `AUTH_MODE=jwt` (JWKS RS256) belum diuji end-to-end; hanya unit test verifier. Jalur aktif kini `AUTH_MODE=session` (auth lokal, ISSUE-15) dan sudah diuji penuh — lihat §9.
- Rate limit in-process, belum diuji pada multi-replica.
- Belum ada CI yang menjalankan suite E2E otomatis; `npm run test:e2e` masih dijalankan manual.
- Dataset masih kecil (puluhan transaksi); benchmark performa dataset realistis belum dilakukan.

## 8. Audit a11y & responsivitas (ISSUE-21)

Referensi visual: `design-refs/` (sudah **dihapus** sesuai permintaan; isi awalnya 19–22 screen dari project Stitch `2264071121943649738`, ditarik lewat REST dengan `X-Goog-Api-Key` karena `sp_screens` MCP masih gagal — bug lama di plan.md §15; bisa ditarik ulang dengan `node scripts/pull-stitch-screens.cjs` lalu `node scripts/unwrap-stitch-screenshots.cjs`). Artefak HTML hasil render: `audit-html/`.

| Pemeriksaan | Hasil |
|---|---|
| Lighthouse (axe) a11y, 8 halaman @1280 | **semua 1.0, nol kegagalan** |
| Sebelum perbaikan | `color-contrast` di 6 halaman (0,96) + `landmark-one-main` di `/login` (0,93) |
| Stitch `sp_a11y` (login & savings) | `warnings: []` — mencakup pemeriksaan touch target |
| Stitch `sp_analyze` | `hasViewport`, `hasBreakpoints`, `hasAccessibility`, `hasTailwind` = true |
| CDP 360px, 8 halaman | `scrollWidth=360`, nol elemen keluar viewport |

Temuan dan perbaikan:
1. `text-muted-dim` (`#64748B`) hanya 3,0–4,1:1 di semua permukaan gelap → diganti `#8493AB` (4,61 terburuk). Titik gagal pertama: hint label 11px di `/savings`.
2. `danger` (`#F43F5E`) 4,16:1 di chip badge ber-tint → diganti `#F66981` (4,60 terburuk); sekaligus memperbaiki Alert error (4,42) dan teks sisa budget negatif.
3. `/login` tidak punya landmark `<main>` → gagal `landmark-one-main`; konten kini dibungkus `<main>`.
4. Kedua deviasi warna dicatat di `plan.md` §17.2 — DESIGN.md tidak diubah diam-diam.

Reproduksi:
```bash
node scripts/capture-audit-html.cjs   # render 8 halaman @360 & 1280 → audit-html/
node scripts/pull-stitch-screens.cjs  # tarik ulang screen Stitch → design-refs/
```

## 9. Auth lokal (ISSUE-15)

`AUTH_MODE=session`: FISCA menerbitkan token sesi acak, menyimpan hash SHA-256 di tabel `sessions`, dan mencabutnya saat keluar. Kata sandi di-hash Argon2id (64 MiB, t=3, p=4). `AUTH_MODE=dev` dihapus total.

| Pemeriksaan | Hasil |
|---|---|
| `go build` / `go vet` / `go test ./...` (dengan `TEST_DATABASE_URL`) | **semua hijau** |
| Migration `0003_auth.sql` | terpasang, goose → versi 3 |
| Test alur auth baru (`auth_flow_test.go`) | **9/9 PASS**, bukan skip |
| `npm run lint`, `tsc --noEmit`, `npm run build` | hijau |
| `npm run test:e2e` | **24/24** — kini lewat `POST /auth/register` sungguhan |
| `npm run smoke:api` | lulus (exit 0, `DONE`) |
| Rate limit auth (AC #7) | **GAGAL lalu diperbaiki** — lihat bawah |
| Verifikasi browser | `/login` tanpa field token; login → `/home`; Keluar → `/login` + token lama 401 di server; tanpa token terkunci; token tak sah → pesan kedaluwarsa |

Temuan:
1. **AC #7 awalnya gagal total**: 30 login salah berturut-turut semuanya 401, nol 429. Penyebabnya Argon2id membatasi throughput ke ~10 percobaan/detik dan `RATE_LIMIT_RPS` juga 10 — konsumsi dan refill seimbang, jadi bucket tidak pernah kosong. Diperbaiki di `server.go`: bucket auth = `RPS/4` (min 2/detik, burst min 5). Sesudahnya 429 muncul pada percobaan ke-5, dan bucket v1 terbukti terpisah (`GET /me` tetap 200 setelah auth dihantam).
2. **`.env` ber-BOM UTF-8** (`EF BB BF`) → `godotenv` menolak diam-diam, `db:migrate`/`db:seed` dari host gagal "DATABASE_URL is required". BOM dibuang. *Kesalahan penulisan file di sesi sebelumnya.*
3. **`.env` masih `AUTH_MODE=dev`** → API gagal start (`dependency api failed to start`) karena config kini menolak nilai tak dikenal. Diganti `session` di `.env` dan `.env.example`.
4. **Teardown e2e mentok di FK** `budget_categories.category_id` (`ON DELETE RESTRICT`) setelah akun smoke ikut memakai domain `@fisca.test`. Pola persempit ke `e2e-%@fisca.test`.
5. Verifikasi jalur login **tidak tercakup e2e** — test menulis token langsung ke `localStorage` — jadi dilakukan manual di browser. Perlu dicatat bila kelak ada test khusus form login.

Reproduksi:
```bash
npm run db:migrate
cd apps/api && TEST_DATABASE_URL=postgres://fisca:fisca@localhost:5434/fisca?sslmode=disable go test ./... -count=1
npm run test:e2e && npm run smoke:api
```

## 10. Form masuk & daftar (ISSUE-22)

Mata password per field, konfirmasi sandi, dan nama depan/belakang. **Backend tidak diubah** — `registerRequest.DisplayName` sudah ada sejak ISSUE-15; frontend menggabungkan nama.

| Pemeriksaan | Hasil |
|---|---|
| Struktur `/login` mode daftar | 5 field, kedua sandi `type="password"`, 2 tombol mata 44×44 ber-`aria-label`/`aria-pressed` |
| Toggle mata | Field pertama → `text` + `aria-pressed=true`; field kedua tetap `password` (**independen**) |
| Validasi klien | Tidak sama, nama depan kosong, sandi < 8 → ketiganya diblokir sebelum request terkirim |
| Sapaan | Daftar "Raditya Pratama" → **"Halo, Raditya 👋"**; `/profile` menampilkan nama lengkap |
| Nama belakang opsional | Daftar tanpa nama belakang → **"Halo, Sri"** |
| Regresi | `lint`/`tsc`/`build` = 0 · **e2e 34/34** · `go test ./...` semua `ok` · smoke `DONE` |
| Lighthouse a11y `/login` | **1.0 di kedua mode** (masuk & daftar), nol kegagalan |
| **Test otomatis form** | `tests/e2e/auth-form.spec.ts` — 5 test × 2 project = **10 run, lulus** |

Cara menguji form ini (semua temuan waktu verifikasi):
1. **`wait load` terlalu cepat** — halaman masih `innerText: ""` (React belum render). Tunggu konten, misalnya teks "FISCA".
2. **Klik yang kalah cepat dari hydrasi gagal diam-diam.** `?.click()` tetap mengembalikan nilai sehingga assert terlihat lulus padahal mode tidak pernah berganti — ini penyebab kegagalan assert pertama.
3. **Label memakai `uppercase`**, jadi `innerText` berbunyi "ULANGI KATA SANDI"; carian teks berhuruf kecil tidak akan cocok.
4. **Snapshot accessibility menampilkan input sandi sebagai `[generic]`** → tidak ada ref untuk `fill_form`; isi lewat setter native React + event `input`.
5. **Docker Desktop engine berhenti** di tengah sesi (halaman kosong, `curl` `status=000`, tak ada proses Docker). Dinyalakan ulang dari `C:\Users\zimig\AppData\Local\Programs\DockerDesktop\Docker Desktop.exe`, stack dijalankan kembali, seluruh QA diulang. Bukan bug kode.

**Test otomatis** (10 Nov 2026): `tests/e2e/auth-form.spec.ts` — alur daftar → sapaan → keluar → masuk (termasuk verifikasi `displayName` di server), tiga validasi klien yang memblokir request, independensi tombol mata + target sentuh 44 px, penolakan sandi salah dari server, dan email terdaftar. **10 run lulus**; suite penuh **34/34**.

**Batasan:** ~~mata password mode masuk belum diuji; pesan token kedaluwarsa masih manual; `tests/` tidak pernah di-typecheck.~~ **Ketiganya ditutup oleh ISSUE-25 (13 Nov 2026)** — `auth-form.spec.ts` kini menguji mata password mode masuk dan pesan "Sesi Anda sudah berakhir", serta `npm run test:types` menjadikan typecheck sebagai rutinitas.

## 11. Penyelarasan design Dark Carbon Glass (ISSUE-23)

Palet Stitch diterapkan tanpa mengorbankan WCAG AA; kontras dihitung deterministik sebelum menerapkan, bukan ditebak.

| Pemeriksaan | Hasil |
|---|---|
| Rasio kontras token baru | `primary` 7,20 (nilai lama 4,84) · `ai` 7,23 · `analytics` 7,20 — semua lulus |
| `primary-hover` / `primary-deep` | **Lama gagal** 3,50 / 2,41 → kini 6,25 / 4,96 (tidak tertangkap Lighthouse karena hanya mengaudit keadaan diam) |
| Badge tiga tone | `statusForPercent` + tone `info`; bar memakai helper yang sama sehingga mustahil berbeda warna |
| Tile baris transaksi | DOM: 48×48 px, radius 16px (percobaan awal `rounded-xl` justru bulat penuh — `--radius-xl` = 24px) |
| `DistributionBars` | Pill `Uncategorized 87%` / `Belanja 13%`; pill "Sisa" absen karena sudah 100% |
| Lighthouse a11y 8 halaman | **1.0** setelah setiap perubahan |
| e2e | 24/24 saat pengerjaan; suite kini **34/34** |

Yang sengaja **tidak** dilakukan: layout struktural Stitch (header lonceng, chip "Low-Carbon Mode", "SYNC AKTIF", tab ganda) tidak disamakan — sebagian elemen karbon/ESG dilarang plan M7. Yang disamakan hanya bahasa visual.

**Batasan:** threshold tiga tone kini punya test e2e (`budget.spec.ts` → *"budget badge renders all three tones"*, 3 kasus × 2 viewport). ~~`DistributionBars` masih tanpa test langsung.~~ **Ditutup oleh ISSUE-25 (13 Nov 2026)**: `onboarding.spec.ts` → *"home distribution pills match the API"* memastikan pill persentase sama dengan yang dikembalikan API.

Reproduksi pemeriksaan DOM (devtools, di `/history`):
```js
[...document.querySelectorAll('li span[class*="place-items-center"]')].map(t => ({
  w: t.offsetWidth, h: t.offsetHeight, r: getComputedStyle(t).borderRadius
}))
```

## 12. Reset password lokal (ISSUE-24) + revisi DESIGN.md

| Pemeriksaan | Hasil |
|---|---|
| `go build` / `go vet` / `go test ./...` (dengan DB) | semua `ok` |
| Test reset password baru | **5/5 PASS** — pencabutan semua sesi, penolakan sandi lama, email tak dikenal, normalisasi email, salt acak, isolasi antar-akun |
| Verifikasi CLI nyata (`npm run auth:reset`) | **7/7 LULUS** |
| `lint` web | 0 |
| `npm run test:e2e` | **36/36** |
| Container | healthy ×3, `api=200`, `web=200` |

**Temuan:**

1. **Token lama diuji lewat verifier produksi**, bukan hanya dengan membaca kolom `revoked_at` — kalau tidak, test hanya membuktikan data terisi, bukan perilaku.
2. **Tool reset sengaja tidak memakai `config.Load()`** dan hanya membaca `DATABASE_URL`. Alat pemulihan tidak boleh gagal karena masalah konfigurasi lain (mis. `AUTH_MODE` tak dikenal) tepat saat pengguna terkunci.
3. **Argon2id dihitung di luar transaksi** — sama seperti `Register` — supaya derivasi ~100ms tidak menahan kunci baris di database.
4. **DESIGN.md §4.1 direvisi** mengikuti 5 deviasi warna, dengan change log di `DESIGN.md` §1.3 (syarat perubahan dokumen `LOCKED`). Kode dan DESIGN.md kini tidak lagi berselisih; rincian di `plan.md` §17.2.

Reproduksi:
```bash
npm run auth:reset -- anda@example.com kata-sandi-baru   # atau lewat stdin
cd apps/api && TEST_DATABASE_URL=postgres://fisca:fisca@localhost:5434/fisca?sslmode=disable \
  go test ./internal/service/ -run ResetPassword -count=1 -v
```

## 13. Cakupan verifikasi menyeluruh (ISSUE-25)

Menutup enam celah yang memang sudah tercatat di dokumen, bukan celah yang ditebak.

| Pemeriksaan | Hasil |
|---|---|
| `npm run test:types` (baru) | **0 error** — sekaligus membuktikan gunanya (lihat bawah) |
| `npm run test:e2e` | **50/50** (sebelumnya 36; +7 definisi test × 2 viewport) |
| Cakupan halaman | `/savings` dan `/profile` kini tercakup → **8 dari 8 halaman** |
| Regresi | tidak ada perubahan kode Go/Next; hanya `package.json`, `tsconfig.test.json`, `tests/` |

**`test:types` menemukan 8 kesalahan tipe nyata sejak percobaan pertama** — 3 di test lama (`clarification.spec.ts` ×2, `isolation.spec.ts`), 5 di test baru yang sedang ditulis. Semua tertangkap sebelum masuk.

Dua kesalahan yang saya perbaiki dari asumsi saya sendiri saat menulis test:

1. **`/purchase-advisor` adalah GET dengan query param `?price=`**, bukan POST body — diverifikasi dari `docs/api/openapi.yaml` sebelum test gagal.
2. Test pertama `/profile` gagal karena `getByText(displayName)` cocok ke 2 elemen: untuk akun e2e, `displayName` adalah bagian sebelum `@` pada email, sehingga email mengandungnya sebagai substring. Diperbaiki dengan `{ exact: true }`.

**Cakupan halaman sesudah issue ini:**

```
/assistant · /budget · /home · /history · /login · /profile · /reports · /savings
= 8 dari 8
```

**Batasan yang tersisa:** `AUTH_MODE=jwt` dan jalur AI/STT (menunggu kredensial, ISSUE-16); rate limit in-process pada multi-replica; celah **fitur** di ISSUE-09/12/13/14 sengaja tidak dimasukkan — itu pekerjaan parsing, bukan verifikasi.

Reproduksi:
```bash
npm run test:types && npm run test:e2e
```

## 14. Penyelarasan aturan antislop (ISSUE-26)

Dosis diturunkan ke batas aturan, bukan fitur dihapus. `lint`/`tsc`/`build`/`test:types` = 0 · **e2e 50/50** · **Lighthouse a11y 8/8 = 1.0**.

| Yang diubah | Sebelum | Sesudah | Cara diukur |
|---|---|---|---|
| Blur (R-10) | 6 permukaan | **2**: `.glass-panel` + bottom navbar | Audit runtime `getComputedStyle().backdropFilter` di `/home` dan `/assistant` |
| Glow (R-13) | 3 jenis, 5 titik | **1 jenis** (`shadow-glow-primary`) | Audit runtime `boxShadow`; grep seluruh `apps/web` |
| Ikon `Sparkles` (R-04) | 14 lokasi | **0** | grep `*.ts` + `*.tsx` |
| Em dash teks produk (R-02) | 4 kalimat + 6 placeholder | **0** | Pemeriksaan khusus, dua cakupan terpisah |

**Reproduksi audit runtime** (devtools, di halaman mana pun):
```js
[...document.querySelectorAll("*")].filter(el => {
  const b = getComputedStyle(el).backdropFilter;
  return b && b !== "none";
}).length
```

**Temuan:**

1. **Screenshot pertama menampilkan halaman yang salah.** `Page.navigate` async sedangkan loop membaca `innerText` sebelum navigasi selesai, sehingga ia puas dengan konten halaman sebelumnya. Diperbaiki dengan menunggu `location.pathname` cocok, lalu diverifikasi lewat log isi DOM.
2. **Dua keputusan pemilik terkait R-02** dicatat: placeholder `"—"` → en dash, dan 225 em dash dokumentasi dibiarkan (dianggap di luar cakupan karena aturannya menyasar copy produk).
3. **DESIGN.md §4.5/§4.6 direvisi** dengan change log §1.3 — dan justru §4.5 sendiri sudah berbunyi *"Use glassmorphism selectively"*, jadi implementasi lama melampaui dokumennya.

**Batasan:** 8 komentar kode masih memuat em dash (diungkapkan di ISSUE-26, bukan disembunyikan); tidak ada test khusus untuk perubahan dosis karena test yang mengaitkan passing dengan styling akan rapuh.

## 15. Regresi adaptasi desktop (22 November 2026)

Run QA penuh setelah enam view diadaptasi ke layout desktop dari desain Stitch (2560px) — lihat worklog adaptasi. Lingkungan sama seperti kop dokumen ini; e2e kini berjalan terhadap **stack produksi `:3000`** (image web sudah direbuild dengan kode baru).

| Pemeriksaan | Hasil |
|---|---|
| `go build` / `go vet` / `go test ./...` (dengan DB) | **semua hijau** |
| `npm run lint:web` · `npm run test:types` · `npm run build:web` | **0 · 0 · lolos** (12 route statis) |
| `npm run stack:config` | env terisi (`NEXT_PUBLIC_API_BASE_URL`, `RATE_LIMIT_BURST`, `OPENAI_MODEL`, `AI_TIMEOUT_SECONDS`) |
| `npm run smoke:api` | **DONE (exit 0)** — idempotency, `BUDGET_WARNING`, proposal approve, kartu asisten heuristic, isolasi cross-user, void |
| `npm run test:e2e` | **50/50 PASS** (desktop1280 + mobile360, terhadap `:3000`) — cakupan8 halaman utuh |
| Rate limit `GET /me` ×30 | 22× `200` + 8× `429`, `Retry-After: 1` |
| Overflow mobile360px,8 halaman | **8/8 PASS** — `scrollWidth=360/360`, nol elemen keluar viewport |
| Lighthouse a11y,8 halaman | **1.0 semua, nol kegagalan** (`/login` diaudit dalam kondisi tanpa token) |
| Dosis R-10 (runtime, `getComputedStyle`) | **2 jenis** `backdrop-filter`: `blur(16px)` panel kartu + `blur(24px)` bottom nav |
| Dosis R-13 (runtime) | **1 jenis** glow emerald (`shadow-glow-primary`) |
| R-04 `Sparkles` · R-02 em dash di source web | **0** · **0** |

**Temuan:**

1. **Dua kegagalan e2e pertama di `onboarding.spec.ts`** — `getByText("Total saldo")` jadi cocok ke2 elemen (hero mobile + strip KPI desktop) karena label yang sama kini hadir sebagai dua varian responsif. Diperbaiki: assert elemen yang **terlihat** (`locator("visible=true")`), niat test (saldo tampil) dipertahankan. Lulus di kedua viewport.
2. **CORS menolak origin `:3001`** saat dev server lokal dipakai untuk verifikasi → `CORS_ORIGINS` di `.env` lokal ditambah `http://localhost:3001` (origin dev; `:3000` tetap utuh). Bukan perubahan kode.
3. **Hydration mismatch tetap teramati** di console dev untuk semua halaman ber-token. **Pra-ada** — build lama di `:3000` melempar React #418 yang sama; akar masalahnya cabang `hasToken()` (server tanpa token vs klien ber-token), di luar cakupan perubahan desktop. Tercatat sebagai risiko lanjutan, bukan regresi.
4. Dosis desktop tidak menambah permukaan baru: kartu KPI/rail memakai level Panel yang sama; kanvas asisten dibuat solid (nol blur di `/assistant`).

**Reproduksi:**
```bash
go test ./... -count=1            # dari apps/api, dengan TEST_DATABASE_URL
npm run lint:web && npm run test:types && npm run build:web
npm run smoke:api
npm run test:e2e                  # stack harus sehat: npm run stack:up
node tests/qa-overflow-dose.mjs   # overflow360 + hitungan dose runtime
node tests/qa-dose-distinct.mjs   # jumlah jenis backdrop-filter & glow unik
```
