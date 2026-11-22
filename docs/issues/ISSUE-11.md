# ISSUE-11 — Login/register, mode JWT, dan logout

**Status:** CLOSED (QA PASS, dengan catatan di bawah)

**Sumber:** PRD §23 exit #1 ("User dapat register/login dan data antar-user terisolasi"), §4.1 Must Have "Authentication dan user profile", plan.md O-04/D-01 (Supabase Auth + verifikasi JWT di Go).

**Acceptance criteria**
1. Halaman `/login` menawarkan masuk dan daftar lewat email + kata sandi bila penyedia autentikasi dikonfigurasi.
2. Tanpa penyedia, halaman jatuh ke token lokal dan menyatakan status itu secara eksplisit; tidak pernah berpura-pura ada login production.
3. Token hasil masuk tersimpan di browser dan dikirim sebagai `Authorization: Bearer`, lalu kedaluwarsa otomatis di-refresh sebelum dipakai.
4. Logout menghapus kredensial lokal dan menghalangi token demo bawaan build masuk kembali.
5. Token tidak valid/kedaluwarsa memunculkan pesan "Sesi Anda sudah berakhir" dan tautan kembali ke `/login`.
6. Mode `AUTH_MODE=jwt` membuktikan: token RS256 valid diterima, token lain ditolak 401 (dev, kedaluwarsa, issuer salah, audience salah, kunci tidak dikenal, kebingungan algoritma HS256, subject bukan UUID).
7. `APP_ENV=production` + `AUTH_MODE=dev` tetap ditolak oleh config (guard yang sudah ada, dipertahankan). *Catatan (10 Nov 2026): guard ini menjadi tidak relevan setelah ISSUE-15 — `AUTH_MODE=dev` dihapus total, jadi tidak ada mode dev untuk ditolak.*

**Implementasi**
- `internal/auth/testjwks`: membuat JWKS lokal + menandatangani token, sehingga jalur JWT bisa diuji tanpa project Supabase sungguhan.
- `internal/auth/jwks_test.go`: 1 kasus diterima + 8 kasus ditolak.
- `internal/server/integration_test.go`: `TestJWTModeAcceptsSignedTokenAndRejectsOthers` — router penuh dalam mode `jwt`, `/me` mengembalikan identitas dari token, dan 4 token tak terpercaya ditolak.
- Web: `lib/session.ts` (penyimpanan token, Supabase GoTrue lewat REST tanpa library tambahan, refresh, `useToken` berbasis `useSyncExternalStore`), `app/login/page.tsx` + `components/views/login-view.tsx`, tombol **Keluar** di Profil, tautan `/login` pada semua halaman tanpa token, `errorMessage` untuk 401.
- Docker: `NEXT_PUBLIC_SUPABASE_URL` / `NEXT_PUBLIC_SUPABASE_ANON_KEY` menjadi build arg.

**QA**
- `go test ./... -count=1` dengan `TEST_DATABASE_URL`: semua paket `ok` (11 test auth + 10 test integrasi).
- `go vet`, `gofmt -l`, `tsc --noEmit`, `npm run lint`, `npm run build`: bersih; rute `/login` ter-generate.
- Browser: buka `/login` → tanpa console error; klik "Masuk dengan token lokal" → `/home` + token tersimpan; Profil → **Keluar** → `/login`, localStorage hanya berisi `fisca.signedOut`, `/home` menampilkan "Belum ada token akses" dengan tautan `/login`; masuk ulang → `/home` pulih.
- CDP viewport 360px: `scrollWidth=360` untuk 7 halaman termasuk `/login`.
- `npm run smoke:api` tetap lulus, `otherCount=0`.

**Temuan QA selama pengerjaan**
- Setelah logout, token demo dari `NEXT_PUBLIC_DEV_TOKEN` menghidupkan kembali sesi sehingga logout terasa tidak berfungsi. Diperbaiki dengan penanda `fisca.signedOut` yang mengalahkan token build; masuk ulang menghapus penanda itu.
- Lint React 19 menolak `setState` di dalam effect untuk membaca token; diganti `useSyncExternalStore` (`useToken`) dan input berbasis `ref` + `defaultValue`.

**Batasan (jujur)**
- Jalur Supabase sungguhan (signup, konfirmasi email, refresh token, logout provider) **belum teruji terhadap project nyata** karena tidak ada kredensial Supabase di lingkungan ini. Yang teruji adalah: kontrak GoTrue ditulis sesuai dokumentasi REST-nya, fallback tanpa penyedia, dan mode `jwt` di API terbukti lewat JWKS lokal.
- Token kedaluwarsa yang gagal di-refresh menghapus kredensial lokal dan mengarahkan ke `/login`; belum ada pengalaman "perpanjang sesi diam-diam" bila jaringan sedang putus saat refresh.
- Verifikasi email, reset password, dan proteksi rate limit khusus endpoint auth belum dikerjakan (GoTrue yang menanganinya).
