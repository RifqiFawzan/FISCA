# ISSUE-15 — Auth lokal: email + kata sandi

**Status:** CLOSED (QA PASS) — selesai 10 Nov 2026. Tidak butuh kredensial eksternal.

**Konteks & keputusan user (10 Nov 2026)**

User menegaskan arah produk: **FISCA dipakai untuk pribadi dulu**; kemungkinan jadi proyek SaaS hanya "nanti jika dirasa sudah saatnya". Konsekuensinya:

1. **Supabase tidak dipakai untuk tahap ini.** Alasan: komitmen ke SaaS belum pasti, dan login tidak perlu bergantung layanan pihak ketiga untuk pemakaian tunggal.
2. Ini **meng-approve D-01 dengan cabang alternatifnya** (`plan.md:740`): *"Bila pilih custom: tambah tabel `sessions`, Argon2id, reset-password flow, rate limit auth"* — bukan cabang Supabase-nya. `O-04` (`plan.md:69`) ikut berubah dari Supabase ke custom auth Go.
3. **Kode Supabase tidak dihapus** — `lib/session.ts` memanggil GoTrue langsung dan `login-view.tsx` sudah punya form email+sandi. Kode itu dibiarkan dorman sebagai jalur cadangan bila nanti jadi SaaS; menghapusnya hanya menambah pekerjaan ulang. Ini diketahui sebagai *dead code* yang tidak pernah teruji.
4. **`AUTH_MODE=dev` dihapus total** (keputusan user, 10 Nov 2026) — **satu jalur auth saja**. Alasan:
   - **Backdoor yang diketahui publik.** `seed/dev_user.sql` menyisipkan user `11111111-1111-4111-8111-111111111111`, dan token untuk user itu dicetak di tiga tempat: placeholder `login-view.tsx:175`, **nilai default tombol** `login-view.tsx:183` (klik "Masuk" tanpa mengetik apa pun langsung masuk), dan `README.md:84`.
   - `dev` adalah **default di mana-mana**: `config.go:63` (`env("AUTH_MODE", AuthModeDev)`) dan `docker-compose.yml:56` (`${AUTH_MODE:-dev}`).
   - Guard `APP_ENV=production` memang menolak `dev` (`config.go:91`), tapi `docker-compose.yml:42` defaultnya `development` — jadi guard **tidak aktif kecuali diatur eksplisit**.
   - **Dua jalur berarti satu tidak pernah diuji.** Dengan satu jalur, seluruh suite melewatinya.

**Posisi hari ini**

- Backend FISCA punya **nol** infrastruktur login: tidak ada endpoint `/auth`, tidak ada hashing kata sandi, tidak ada kolom password. Yang ada hanya `auth.Verifier` yang mempercayai token.
- `dev` mode: `dev:<uuid>` dipercaya apa adanya (`verifier.go:52`) — bukan autentikasi, dan **dihapus di issue ini** (lihat konteks #4).
- `jwt` mode (`verifier.go:72`) menunggu Supabase; tidak dipakai di tahap ini, dibiarkan dorman.
- Web sudah punya form email + kata sandi, tapi menunjuk ke GoTrue (`session.ts:191`).

**Desain yang disarankan**

| Aspek | Keputusan | Alasan |
|---|---|---|
| Hash kata sandi | **Argon2id** (`golang.org/x/crypto/argon2`) | Sesuai `plan.md:740`; paket sudah ada sebagai indirect dependency (`v0.58.0`), jadi tidak menambah dependensi unduhan |
| Sesi | Tabel **`sessions`** (migration `0003_auth.sql`) + **hash token**, bukan token mentah | Bila DB bocor, token tidak bisa dipakai langsung. Sama prinsipnya dengan kata sandi |
| Token | Acak 32 byte → disimpan ter-hash, masa berlaku panjang (mis. 30 hari) | Pemakaian pribadi tidak butuh rotasi refresh yang rumit; dan **dapat dicabut** saat keluar |
| Verifier | Tipe baru yang **lookup ke DB** (implements `auth.Verifier`) | Muat di arsitektur `cmd/api/main.go:84` yang sudah ada |
| Endpoint | `POST /auth/register`, `POST /auth/login`, `POST /auth/logout` | Tiga ini sudah cukup untuk AC |
| Mode | **Hapus `AUTH_MODE=dev`**; `AUTH_MODE` tinggal `session` (default, baru) dan `jwt` (Supabase, dorman) | Dihapus atas keputusan user (konteks #4): satu jalur auth, hilangkan backdoor placeholder |
| Transport web | Ganti `gotTrue()` → panggil API FISCA di `session.ts` | **UI tidak berubah** — form email/sandi sudah ada |
| `NEXT_PUBLIC_DEV_TOKEN` | **Dihapus** dari `.env`, `session.ts`, `Dockerfile` arg, dan `compose` | Tanpa auth lokal ia bikin browser langsung login sebagai user dev |
| Field "Token lokal" di `/login` | **Dihapus** | Dengan auth lokal, form email+sandi selalu tampil; jalur tempel token tidak lagi relevan |
| e2e | `freshToken()` → **daftar lewat API** di `tests/e2e/helpers.ts` | Terkonsentrasi di satu file; `openAs()` tetap. Test kini melewati jalur auth sungguhan |
| Seed | `dev_user.sql` diberi **kata sandi** (atau diganti jadi registrasi) | Tanpa itu tidak ada akun awal yang bisa dipakai untuk masuk |
| Email | **Tanpa konfirmasi email** | Supabase punya; auth lokal tidak. Reset = akses DB langsung |

**Acceptance criteria**
1. Daftar (email + kata sandi) → token sesi diterbitkan → token itu diterima API.
2. Masuk dengan kredensial yang sama → token baru; kata sandi salah → ditolak dengan pesan yang jelas dan **tidak membocorkan apakah email terdaftar**.
3. `/home`, `/history`, `/assistant` menampilkan data user yang benar, terpisah antar akun (isolasi).
4. Keluar: token **dicabut di server**, kembali ke `/login`, halaman lain terkunci, dan token lama ditolak bila dipakai ulang.
5. Token kedaluwarsa → pesan "Sesi Anda sudah berakhir".
6. **Jalur dev sudah hilang:** `AUTH_MODE=dev` tidak lagi dikenal (ditolak config dengan pesan jelas), token `dev:<uuid>` ditolak 401, dan `/login` tidak lagi menampilkan field token lokal maupun placeholder `dev:11111111-…`. Tanpa kredensial yang benar, endpoint terproteksi tetap 401.
7. Rate limit pada endpoint auth (aturan di `plan.md:740`).
8. `npm run lint`, `npm run build`, `npm run test:e2e`, `go test ./...` semua hijau — **e2e lewat jalur auth sungguhan**, bukan mode dev.
9. Catatan hasil ditulis ke `docs/qa.md` dan ISSUE-15 ditutup.

**Di luar cakupan**
- **Reset-password lewat UI** — `plan.md:740` menyebutnya sebagai konsekuensi custom auth, tapi untuk pemakaian pribadi pemulihan cukup lewat akses DB. Dicatat di `docs/backlog.md` sebagai pekerjaan terpisah; jangan hilang.
- **Konfirmasi email, OAuth, MFA, multi-tenant** — tidak relevan untuk pemakaian pribadi.
- **Menghapus kode Supabase** — sengaja dipertahankan (lihat konteks #3).

**Batasan risiko**
- Auth menyentuh kode keamanan. Kebocoran di sini berarti seluruh data pengguna. Verifikasi manual di AC #2 dan #4 wajib, bukan hanya test otomatis.
- Tidak ada test yang khusus menguji kekuatan hash Argon2id — keputusan parameter (time, memory, parallelism) harus dicatat di `docs/qa.md` agar bisa ditinjau ulang.

**Temuan riset yang tetap relevan** (hasil penyelidikan jalur Supabase, 10 Nov 2026)
- `NEXT_PUBLIC_DEV_TOKEN` **tidak menopang e2e** — test menulis token langsung ke `localStorage` lewat `openAs()` (`tests/e2e/helpers.ts`). Itulah alasan **menghapusnya seluruhnya aman** (konteks #4), bukan sekadar mengosongkan.
- `stack:up` memakai `--env-file .env` (root), jadi cukup satu file.
- `/me` meneruskan email kosong (`internal/api/handler.go:104`), sehingga user baru tersimpan `<uuid>@fisca.local`. **Harus diperbaiki di issue ini** — dengan auth lokal, email asli sudah diketahui saat register dan harus ikut tersimpan.

**QA (10 Nov 2026)**

- Migration `0003_auth.sql` terpasang (goose → versi 3); `sqlc generate` menghasilkan `generated/auth.sql.go` (6 query).
- `go build`, `go vet`, `go test ./...` dengan `TEST_DATABASE_URL` — semua hijau.
- **9 test alur baru** (`internal/server/auth_flow_test.go`) lulus, semuanya berjalan bukan skip: register → token diterima `/me`; pengguna baru dapat akun & kategori bawaan; email tak dikenal / tanpa kata sandi / kata sandi salah semuanya **pesan identik**; duplikat → 409; input tidak valid → 422; logout mencabut → token lama 401; sesi kadaluwarsa → 401; token `dev:` → 401.
- Unit `internal/auth`: round-trip Argon2id, salt unik, hash terformat salah & parameter di luar batas ditolak, `ValidatePassword`, token 43 karakter URL-safe, digest SHA-256 deterministik.
- `npm run lint`, `tsc --noEmit`, `npm run build` hijau.
- `npm run test:e2e` **24/24** — kini tiap test mendaftar lewat `POST /auth/register` sungguhan, bukan token khusus test.
- `npm run smoke:api` lulus (exit 0, `DONE`).
- **AC #7 terukur GAGAL pada percobaan pertama**, lalu diperbaiki: 30 login salah berturut-turut semuanya 401, nol 429, karena Argon2id membatasi throughput ke ~10/detik — persis `RATE_LIMIT_RPS`. Bucket auth diubah ke `RPS/4` (min 2/detik, burst min 5); sesudahnya 429 muncul pada percobaan ke-5 dan bucket v1 tetap utuh (`GET /me` tetap 200). Terukur di `server.go`.
- **Verifikasi browser** (jalur yang tidak dijangkau e2e, karena test menulis token ke `localStorage`): `/login` hanya menampilkan email + kata sandi — nol field token, nol placeholder `dev:`, nol peringatan Supabase; login lewat form → `/home` menampilkan "Halo, smoke-a"; `/profile` menampilkan email & zona waktu yang benar; **Keluar** → `/login`, `localStorage` kosong, dan token lama ditolak API dengan 401 (pencabutan di server, bukan hanya di klien); tanpa token `/home` terkunci "Belum ada token akses"; token tidak sah → **"Sesi Anda sudah berakhir. Silakan masuk kembali."**

**Temuan QA**
1. **`.env` terkena BOM UTF-8** (`EF BB BF`) — akibat penulisan ulang oleh PowerShell `-Encoding utf8` saat menyimpan Stitch key sebelumnya. `godotenv` menolak diam-diam (`envfile.Load` mengabaikan galatnya), sehingga `db:migrate` dan `db:seed` dari host gagal dengan "DATABASE_URL is required". BOM dibuang; file ditulis ulang tanpa BOM. *Ini kesalahan saya sendiri, bukan kode awal.*
2. **`.env` masih berisi `AUTH_MODE=dev`** → API gagal start (`dependency api failed to start`) karena config kini menolak nilai tak dikenal. Diganti `AUTH_MODE=session` di `.env` dan `.env.example`.
3. **Teardown e2e mentok di FK**: `budget_categories.category_id` adalah `ON DELETE RESTRICT`, sehingga menghapus user yang kategorinya dialokasikan ke budget gagal. Pola dibuat sempit ke `e2e-%@fisca.test` — akun smoke sengaja tidak disentuh (smoke idempoten dan memang punya alokasi budget).
4. Rate limit auth tidak mengena pada konfigurasi awal (lihat QA di atas).
5. Tool snapshot browser mengembalikan konten terpotong (673 char) di `/profile`, jadi verifikasi tombol **Keluar** dilakukan lewat klik DOM langsung.

**Batasan:** reset-password lewat UI belum ada (tercatat di `docs/backlog.md`); jalur Supabase dorman tidak diuji ulang; rate limit masih in-process per replica (lihat `docs/qa.md` §7).
