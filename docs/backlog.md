# FISCA — Backlog

Sumber: `plan.md` §11 (milestones), `PRD_FISCA.md`, `DESIGN.md`, `workflow.md`.
Status per 8 November 2026. `PASS` berarti acceptance criteria diverifikasi QA (lihat `docs/qa.md`).

**Penomoran issue:** `ISSUE-01` s.d. `ISSUE-26` — masing-masing di berkasnya sendiri di `docs/issues/` (indeks lengkap di `docs/issues.md`) — **25 CLOSED, 0 OPEN, 1 BLOCKED**. **Nomor berikutnya `ISSUE-27`**; issue baru harus dimulai dari 27.

Progres (diurutkan menurut PRD §23 exit criteria lalu §4 Must Have):
- **ISSUE-09** tanggal relatif di asisten — CLOSED.
- **ISSUE-10** integration & security test otomatis (exit #19, #20) — CLOSED.
- **ISSUE-11** login/register + mode JWT + logout (exit #1) — CLOSED.
- **ISSUE-12** multi-transaksi satu kalimat (PRD §22.3) — CLOSED.
- **ISSUE-13** koreksi transaksi via chat — CLOSED.
- **ISSUE-14** tanggal & frasa lanjutan — CLOSED.
- **ISSUE-15** auth lokal: email + kata sandi — **CLOSED (QA PASS, 10 Nov 2026)**. Cakupannya **diganti** dari "verifikasi Supabase" karena user menegaskan FISCA untuk **pribadi dulu**, SaaS hanya mungkin nanti. Meng-approve D-01 cabang custom (`plan.md`): Argon2id + tabel `sessions` + token di-hash server-side + `AUTH_MODE=session`, dan **`AUTH_MODE=dev` dihapus total** (satu jalur auth; menghapus backdoor placeholder `dev:11111111-…` yang dulu tampil di `/login` dan README). 9 test alur auth baru lulus; e2e 24/24 kini lewat `POST /auth/register` sungguhan.
- **ISSUE-16** verifikasi voice/STT end-to-end — BLOCKED (butuh `OPENAI_API_KEY`).
- **ISSUE-17** eval set AI 40 kasus — CLOSED.
- **ISSUE-18** Playwright E2E wajib — CLOSED.
- **ISSUE-19** OpenAPI + codegen DTO — CLOSED.
- **ISSUE-20** Savings Planner & Purchase Advisor — CLOSED (Should Have, terakhir yang mandiri).
- **ISSUE-21** audit a11y & responsivitas UI — CLOSED (Lighthouse 8/8 = 1.0).
- **ISSUE-22** form masuk & daftar (mata password, konfirmasi sandi, nama depan/belakang) — CLOSED (QA PASS). Backend tak berubah; jalur form kini tercakup 10 run Playwright di `auth-form.spec.ts`.
- **ISSUE-23** penyelarasan design Dark Carbon Glass (5 deviasi warna, badge tiga tone, `DistributionBars`, tile baris transaksi) — CLOSED (QA PASS). **Dicatat mundur**: pengerjaannya selesai 10 Nov tanpa issue, jadi baru ditulis 11 Nov. Layout struktural Stitch sengaja tidak disamakan (elemen karbon/ESG dilarang plan M7).
- **ISSUE-24** reset password lokal (CLI `npm run auth:reset`) — CLOSED (QA PASS). Menutup celah yang tercipta saat ISSUE-15 menghapus mode `dev`: lupa sandi dulu berarti membuat hash Argon2id manual. Pemberitahuan "email sudah terdaftar" saat daftar sudah ada sejak ISSUE-22.
- **ISSUE-25** cakupan verifikasi — CLOSED (QA PASS). `npm run test:types` (baru; langsung menemukan 8 kesalahan tipe), e2e `/savings` dan `/profile` (sebelumnya nol), pesan token kedaluwarsa, mata password mode masuk, dan `DistributionBars`. **Seluruh 8 halaman kini tercakup e2e**; suite **50/50**.
- **ISSUE-26** penyelarasan dengan aturan antislop — CLOSED (QA PASS). Dosis glass diturunkan 6 → 2 permukaan, glow 3 jenis → 1, `Sparkles` 14 → 0 (diganti berbasis relevansi), dan teks produk dibebaskan dari em dash. DESIGN.md §4.5/§4.6 direvisi dengan change log. Delivery Gate **PASS (4/4 block)**.

**Status:** **ISSUE-15 CLOSED (QA PASS)** — auth lokal email + kata sandi selesai 10 Nov 2026. **ISSUE-16 masih BLOCKED** (butuh `OPENAI_API_KEY`) dan kini menjadi satu-satunya yang tersisa.

**Fokus saat ini (11 Nov 2026):** tidak ada issue `OPEN`. Penyelarasan design Stitch sudah tercatat (ISSUE-23) dan form masuk/daftar sudah ber-test otomatis (ISSUE-22). Yang tersisa hanya ISSUE-16 yang menunggu `OPENAI_API_KEY`.

| ID | Milestone | Acceptance criteria | Status | Bukti |
|---|---|---|---|---|
| M1 | Foundation | `/health` mengembalikan `status=ok, db=ok`; CORS sesuai daftar origin; Docker Compose menyalakan `db + db-init + api + web`; image non-root; `go build`/`go vet`/`npm run lint`/`npm run build` hijau | **PASS** | [`docs/qa.md`](./qa.md) §1, [`ISSUE-01`](./issues/ISSUE-01.md) |
| M2 | Ledger & data master | Uang integer rupiah (BIGINT); saldo turunan ledger (tidak ada kolom saldo mutable); akun hybrid; kategori default auto-provision; isolasi antar-user; audit trail | **PASS** | [`docs/qa.md`](./qa.md) §2, [`ISSUE-02`](./issues/ISSUE-02.md) |
| M3 | Transaksi & idempotency | `POST /transactions` wajib `Idempotency-Key`; retry tidak menggandakan; key sama + body beda → `409 IDEMPOTENCY_MISMATCH`; PATCH optimistik (`version`); void bukan hard-delete | **PASS** | [`docs/qa.md`](./qa.md) §3, [`ISSUE-03`](./issues/ISSUE-03.md) |
| M4 | Dashboard & analitik | Agregasi hanya di backend; periode half-open `[start, end)` dengan tz pengguna; `sufficientData` pada unusual spending (median+MAD); tidak ada klaim "tersimpan" sebelum server konfirmasi | **PASS** | [`docs/qa.md`](./qa.md) §4, [`ISSUE-04`](./issues/ISSUE-04.md) |
| M5 | Historical intelligence | Pencarian bahasa alami → filter terstruktur; kata sisa tidak mencemari `q`; largest transaction berasal dari query DB | **PASS** | [`docs/qa.md`](./qa.md) §5, [`ISSUE-05`](./issues/ISSUE-05.md) |
| M6 | Budget & advisor | Budget butuh `version`; perubahan lewat proposal + approval (bukan langsung aktif); over-budget hanya warning, transaksi tetap tersimpan; asisten tidak pernah menulis `user_id` | **PASS** | [`docs/qa.md`](./qa.md) §6, [`ISSUE-06`](./issues/ISSUE-06.md) |
| M7 | Pilot hardening | Rate limit per-user + ketat untuk AI/STT; log operasional JSON dengan request id; frontend seluruh halaman tersambung API; responsif 360px & desktop; tidak ada copy carbon/ESG/clean-energy | **PASS** | [`docs/qa.md`](./qa.md) §7, [`ISSUE-07`](./issues/ISSUE-07.md) |

## Di luar cakupan implementasi saat ini (dicatat, tidak diklaim selesai)

- Modul **savings / purchase planning** belum ada endpoint di kontrak `plan.md` §7.2 dan belum diimplementasi.
- **Reset-password lewat UI** belum ada: `plan.md` D-01 menyebutnya sebagai konsekuensi auth lokal, tapi ISSUE-15 sengaja mengeluarkannya dari cakupan. Pemulihan sementara lewat akses DB. Ini pekerjaan terpisah berikutnya bila diperlukan.
- **Jalur Supabase** (`lib/session.ts` blok dorman, `AUTH_MODE=jwt`, `NEXT_PUBLIC_SUPABASE_*`) tidak dihapus dan tidak diuji ulang — dipertahankan sebagai cadangan bila FISCA kelak jadi proyek SaaS (ISSUE-15 konteks #3).
- **STT & intent AI** aktif hanya bila `OPENAI_API_KEY` diisi; tanpa key asisten memakai heuristik deterministik dan `/speech/transcriptions` mengembalikan `503 AI_UNAVAILABLE`.
- **Eval set AI 40 kasus** (`docs/eval/ai-cases.jsonl`), **OpenAPI/codegen DTO**, dan **Playwright component test** belum dibuat; kontrak TypeScript ditulis manual mirror DTO Go dan dijaga oleh smoke test.
- **Multi-transaksi dalam satu kalimat**, **koreksi "ubah kopi tadi jadi 15rb"**, dan **tanggal relatif "kemarin"** belum diparsing; klarifikasi draft sudah bisa dilanjutkan (E2E #3).
- Rate limit bersifat **in-process per replica**; untuk multi-replica perlu shared store (Redis).
- Chart memakai CSS bars, bukan Recharts; TanStack Query / React Hook Form + Zod / shadcn-ui tidak dipakai (lihat `plan.md` §13).
