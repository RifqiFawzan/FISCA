# ISSUE-22 — Form masuk & daftar: mata password, konfirmasi, nama depan/belakang

**Status:** CLOSED (QA PASS) — selesai 10 Nov 2026.

**Sumber:** permintaan user langsung setelah ISSUE-15.

**Acceptance criteria**
1. Field kata sandi pada mode masuk **dan** daftar punya tombol mata; menekannya hanya mengubah field itu sendiri.
2. Daftar meminta konfirmasi kata sandi; sandi yang tidak sama ditolak **sebelum** request terkirim.
3. Daftar meminta nama depan & belakang; nama depan yang dipakai aplikasi menyapa ("Halo, Raditya"); nama belakang opsional.
4. Validasi klien memblokir submit: nama depan wajib, kata sandi ≥ 8 karakter, kecocokan sandi.
5. `lint`, `tsc`, `build`, e2e, dan smoke tetap hijau.

**Implementasi**
- `components/views/login-view.tsx`: komponen lokal `PasswordInput` dengan tombol mata per field — `aria-label` ("Lihat/Sembunyikan kata sandi"), `aria-pressed`, `type="button"` agar tidak ikut meng-submit, dan 44×44 px agar nyaman disentuh. Field nama depan/belakang + konfirmasi sandi hanya pada mode daftar.
- `lib/session.ts`: `signUp(email, password, displayName = "")` meneruskan nama ke API.
- **Backend tidak diubah sama sekali** — `registerRequest.DisplayName` sudah ada sejak ISSUE-15 dan `service.Register` menerimanya; frontend menggabungkan nama depan + belakang. Bila belakang kosong, `displayName` hanya berisi nama depan.
- Sapaan memakai `displayName.split(" ")[0]` (`home-view.tsx:75`), jadi token pertama = nama depan.

**QA (jalur browser — form ini tidak dicakup test otomatis)**
- Struktur: 5 field, kedua input `type="password"`, dua tombol mata 44×44 ber-label "Lihat kata sandi" dengan `aria-pressed="false"`.
- Toggle: field pertama menjadi `text` + label "Sembunyikan kata sandi" + `aria-pressed="true"`, field kedua tetap `password` → **independen per field**.
- Validasi: sandi tidak sama → "Kata sandi tidak sama. Ketik ulang dengan hati-hati."; nama depan kosong → "Nama depan wajib…"; sandi < 8 → "minimal 8 karakter". Ketiganya tetap di `/login`, **tidak ada request terkirim**.
- Sukses: daftar "Raditya Pratama" → `/home`, sapaan **"Halo, Raditya 👋"**, dan `/profile` menampilkan "Raditya Pratama".
- Nama belakang opsional: daftar tanpa nama belakang → sapaan **"Halo, Sri"** ✓.
- Regresi: `lint`/`tsc`/`build` = 0; **e2e 24/24**; `go test ./...` semua `ok`; smoke `DONE`.

**Temuan QA (cara menguji form ini di browser)**
1. `wait load` kembali sebelum React merender — halaman masih `innerText: ""`. Wajib menunggu konten (mis. teks "FISCA") sebelum berinteraksi.
2. Klik yang bersaing dengan hydrasi **gagal diam-diam**: `?.click()` tetap mengembalikan nilai, sehingga test terlihat sukses padahal mode tidak pernah berganti. Ini sumber kegagalan assert pertama saya.
3. Label memakai kelas `uppercase`, jadi `innerText` berbunyi "ULANGI KATA SANDI" — pencarian teks berhuruf kecil tidak akan pernah cocok.
4. Snapshot accessibility menampilkan input kata sandi sebagai `[generic]`, sehingga tidak punya ref untuk `fill_form`; pengisian dilakukan lewat setter native React + event `input`.
5. **Docker Desktop engine berhenti di tengah sesi verifikasi** sehingga halaman kosong dan `curl` `status=000`. Bukan bug kode — engine dinyalakan ulang dan stack dijalankan kembali, lalu seluruh QA diulang dari awal.

**Test otomatis (ditambahkan 10 Nov 2026)** — `tests/e2e/auth-form.spec.ts`, 5 test × 2 project = **10 run**, bagian dari suite **34/34**:
1. Daftar lewat form → sapaan "Halo, Nadia", lalu `GET /me` memastikan `displayName` "Nadia Putri" dan email tersimpan di **server** → Keluar → masuk kembali dengan kredensial sama.
2. Validasi klien memblokir **sebelum** request: sandi tidak sama, nama depan kosong, sandi < 8 — ketiganya tetap di `/login` dan tidak ada token terbit.
3. Tombol mata independen per field (`type` berubah hanya di field yang diklik) + ukuran target sentuh ≥ 44 px.
4. **Penolakan server**: sandi salah pada akun nyata → "Email atau kata sandi salah" (pesan seragam, AC ISSUE-15 #2).
5. Email sudah terdaftar → notice "Email sudah terdaftar. Silakan masuk."

**Batasan:** ~~mata password mode masuk belum diuji sendiri; pesan token kedaluwarsa masih diverifikasi manual.~~ **Keduanya ditutup oleh ISSUE-25 (13 Nov 2026)** — `tests/e2e/auth-form.spec.ts` kini punya test untuk mata password pada mode masuk dan untuk pesan "Sesi Anda sudah berakhir". Tidak ada sisa batasan di issue ini.
