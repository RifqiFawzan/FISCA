# ISSUE-24 — Reset password lokal (CLI)

**Status:** CLOSED (QA PASS) — selesai 12 Nov 2026.

**Sumber:** keputusan user (12 Nov 2026) — *"jalankan CLI reset lokal"*. Di sesi yang sama user juga meminta pemberitahuan saat daftar bila email sudah dipakai; lihat bagian terakhir: **itu sudah ada**.

## Masalah yang ditutup

Saat ISSUE-15 menghapus `AUTH_MODE=dev`, FISCA kehilangan satu-satunya jalur pemulihan. Dulu akun seed bisa diakses cukup dengan menempel `dev:<uuid>`; setelah itu, lupa kata sandi berarti **membuat hash Argon2id secara manual**. Celah ini saya ciptakan sendiri dan sengaja tidak ditutup tanpa keputusan user.

## Acceptance criteria

1. Perintah lokal mengganti kata sandi akun yang ada.
2. **Seluruh** sesi akun itu ikut dicabut.
3. Kata sandi lama ditolak; kata sandi baru diterima.
4. Email tak dikenal dan sandi terlalu pendek memberi pesan yang jelas, bukan 500.
5. Test otomatis lulus, verifikasi CLI nyata lulus, seluruh suite tetap hijau.

## Implementasi

| Bagian | Isi |
|---|---|
| `db/queries/auth.sql` | `UpdateUserPassword` (`:one`) + `RevokeAllSessionsForUser` (`:execrows`) → `sqlc generate` |
| `internal/service/auth.go` | `ResetPassword(email, newPassword)` — validasi → Argon2id **di luar transaksi** (lambat; jangan tahan kunci baris) → UPDATE + cabut sesi **dalam satu transaksi** |
| `cmd/resetpassword/main.go` | `npm run auth:reset -- <email> [kata-sandi]`, atau lewat stdin bila argumen tidak diberikan |
| `package.json` | script `auth:reset` |

Dua keputusan desain yang perlu dicatat:

- **Hanya membaca `DATABASE_URL`, bukan `config.Load()`.** Tool pemulihan tidak boleh gagal karena masalah konfigurasi lain (mis. `AUTH_MODE` tak dikenal) tepat saat pengguna terkunci. Berbeda dengan `cmd/seed` yang menolak berjalan di produksi, **tool ini BOLEH jalan di produksi** — justru itulah gunanya — tetap butuh akses shell ke mesin.
- **Semua sesi dicabut, bukan hanya yang sedang dipakai.** Reset biasanya terjadi karena akun dianggap tidak aman lagi.

## QA

**5 test otomatis baru** di `internal/service/password_reset_test.go` (butuh `TEST_DATABASE_URL`):

1. Reset mengganti sandi **dan** mencabut semua sesi; token lama **ditolak oleh verifier produksi** (bukan sekadar kolom `revoked_at` terisi); sandi lama ditolak dengan pesan seragam; sandi baru diterima.
2. Email tak dikenal → "Email tidak ditemukan"; sandi < 8 ditolak **sebelum** menyentuh database (tidak ada baris tercipta).
3. Email berhuruf besar dan berspasi tetap cocok.
4. Dua reset dengan sandi sama → hash berbeda (salt acak) dan hash tetap valid.
5. Reset satu akun tidak mengubah sesi akun lain.

**Verifikasi CLI nyata — 7/7 lulus** (memakai perintah yang sama dengan pengguna):

```
npm run auth:reset -- <email> <kata-sandi-baru>
  → kata sandi diperbarui untuk <email>
  → sesi dicabut: 1
  → masuk kembali dengan kata sandi yang baru.
```

| Pemeriksaan | Hasil |
|---|---|
| Register akun uji | 201 |
| CLI selesai + melaporkan pencabutan | LULUS |
| Kata sandi lama | **401 ditolak** |
| Kata sandi baru | **200 diterima** |
| Email tak dikenal | `Email tidak ditemukan` |
| Sandi < 8 karakter | `kata sandi minimal 8 karakter` |

**Regresi:** `go build` / `go vet` / `go test ./...` semua `ok` · `lint` 0 · **e2e 36/36** · container healthy.

## Permintaan user yang kedua: pemberitahuan email terdaftar

> *"pada saat daftar beritahu user kalau sudah ada yang menggunakan email tersebut"*

**Sudah berfungsi sejak ISSUE-22.** `lib/session.ts` menerjemahkan `409 CONFLICT` menjadi notice **"Email sudah terdaftar. Silakan masuk."**, dan ditutup test e2e *"sign up points to sign in when the email already exists"*.

Registrasi memang tidak bisa menyembunyikan keberadaan akun — itu bukan kebocoran informasi yang bisa dihindari — jadi desainnya mengarahkan pengguna ke form masuk, bukan menampilkan galat teknis.

## Batasan

- **Reset lewat email tidak ada** — sengaja, sesuai arah "pribadi dulu" tanpa SMTP. Bila kelak jadi SaaS, jalur email perlu ditambahkan; alat CLI ini tetap berguna sebagai jalan cadangan.
- Kata sandi yang ditulis sebagai argumen ikut tersimpan di riwayat shell — bentuk pipa (`echo ... | npm run auth:reset -- <email>`) disediakan untuk menghindarinya.

## Catatan lintas-issue

Di sesi yang sama, **DESIGN.md §4 direvisi** mengikuti 5 deviasi warna yang sudah lama tercatat di `plan.md` §17.2, dengan change log di `DESIGN.md` §1.3 — sesuai aturan `plan.md` bahwa status `LOCKED` tidak boleh diubah tanpa change log. Itu keputusan terpisah, bukan bagian issue ini.
