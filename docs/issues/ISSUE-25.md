# ISSUE-25 — Cakupan verifikasi menyeluruh: typecheck test + e2e tiap halaman + celah tercatat

**Status:** CLOSED (QA PASS) — selesai 13 Nov 2026.

**Sumber:** permintaan user (13 Nov 2026) — "lanjutkan ke issue 25". Nomor berikutnya yang bebas setelah ISSUE-24. Isinya diambil dari **daftar celah yang memang sudah tercatat** di dokumen, bukan dari tebakan.

## Masalah

Verifikasi FISCA sudah kuat (36 e2e, test Go dengan database, smoke, Lighthouse), tetapi ada enam celah yang tertulis sendiri di dokumen:

| # | Celah | Tercatat di |
|---|---|---|
| 1 | `tests/` **tidak pernah di-typecheck** oleh script mana pun (tidak ada tsconfig root) | `docs/qa.md` §10 |
| 2 | Halaman **`/savings` nol test e2e** | `docs/issues/ISSUE-20.md` Batasan |
| 3 | Halaman **`/profile`** hanya dikunjungi untuk menekan "Keluar", tidak diuji isinya | temuan baru (regresi pemeriksaan cakupan) |
| 4 | Pesan **"Sesi Anda sudah berakhir"** di peramban hanya diverifikasi manual | `docs/qa.md` §9, ISSUE-22 Batasan |
| 5 | **Mata password mode masuk** (satu field) belum diuji — yang diuji mode daftar | `docs/qa.md` §10, ISSUE-22 Batasan |
| 6 | **`DistributionBars`** tanpa test langsung | `docs/qa.md` §11, ISSUE-23 Batasan |

Bukti cakupan halaman sebelum issue ini (dihitung dari pemakaian `openAs`/`goto` di `tests/e2e`):

```
/assistant 5x · /budget 2x · /home 2x · /reports 2x · /history 1x · /login 2x
/profile 1x (hanya untuk logout) · /savings 0x
```

## Acceptance criteria

1. `npm run test:types` tersedia dan lulus — seluruh berkas `tests/` ter-typecheck secara **rutin**, bukan sekadar saat saya ingat menjalankannya.
2. `/savings` punya test e2e: halaman termuat, target tabungan tampil, dan alur pembuatan target berfungsi.
3. `/profile` punya test e2e: data akun tampil benar dan aksi keluar berfungsi dari halaman itu.
4. Pesan token kedaluwarsa/tidak sah di peramban **terbukti otomatis** (tidak lagi manual).
5. Mata password pada **mode masuk** teruji (field tunggal), melengkapi mode daftar.
6. `DistributionBars` teruji: pill persentase sesuai data API.
7. Regresi: `lint`, `tsc`, `build`, `go test ./...`, dan e2e tetap hijau.

## QA

| Pemeriksaan | Hasil |
|---|---|
| `npm run test:types` | **0 error** — dan langsung terbukti berguna (lihat bawah) |
| `npm run test:e2e` | **50/50** (sebelumnya 36; +7 definisi test × 2 viewport) |
| Cakupan halaman | `/savings` dan `/profile` kini punya test; **seluruh 8 halaman tercakup** |
| Regresi lain | tidak ada perubahan kode Go/Next di issue ini — hanya `package.json`, `tsconfig.test.json`, dan `tests/` |

**`test:types` menemukan kesalahan nyata sejak percobaan pertama:**

| Lokasi | Kesalahan |
|---|---|
| `clarification.spec.ts:33,34` | `rows[0].description` tanpa penjaga — `Object is possibly 'undefined'` |
| `isolation.spec.ts:15` | pola sama, `rows[0].id` |
| `onboarding.spec.ts`, `savings.spec.ts` | 5 kesalahan serupa di test baru yang sedang saya tulis |

Total **8 kesalahan tertangkap sebelum masuk**. Perbaikannya disengaja berbeda sesuai kegunaan:

- Hanya **dibandingkan** → optional chaining (`rows[0]?.description`), supaya bila assertion kendur kegagalannya terbaca sebagai selisih nilai, bukan `TypeError`.
- Nilainya **dipakai untuk request** → `rows[0]!.id` dengan komentar, karena menggantinya dengan optional chaining akan menghasilkan URL `".../undefined"` yang gagal diam-diam.

**Dua hal yang saya perbaiki dari asumsi saya sendiri saat menulis test:**

1. `/purchase-advisor` ternyata **GET dengan query param** `?price=`, bukan POST body — saya cek `docs/api/openapi.yaml` sebelum test gagal, bukan setelah.
2. Test pertama `/profile` gagal karena `getByText(displayName)` **cocok ke 2 elemen**: untuk akun e2e, `displayName` adalah bagian sebelum `@` pada email, sehingga email mengandungnya sebagai substring. Diperbaiki dengan `{ exact: true }`.

**Cakupan halaman sesudah issue ini** (dihitung dari `openAs`/`goto` di `tests/e2e`):

```
/assistant · /budget · /home · /history · /login · /profile · /reports · /savings
= 8 dari 8 halaman
```

## Batasan yang tetap tersisa

- `AUTH_MODE=jwt` (Supabase) dan jalur AI/STT sungguhan tetap tidak teruji — menunggu kredensial (ISSUE-16, `docs/qa.md` §7).
- Rate limit in-process belum diuji pada multi-replica.
- Celah **fitur** di issue lama (ISSUE-09/12/13/14 — bentuk kalimat & tanggal bahasa Indonesia) sengaja tidak dimasukkan: itu pekerjaan parsing, bukan verifikasi, dan layak issue tersendiri.
