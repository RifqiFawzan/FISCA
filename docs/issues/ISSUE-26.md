# ISSUE-26 — Kurangi dosis glass, glow, dan ikon sesuai aturan antislop

**Status:** CLOSED (QA PASS) — dikerjakan 21 Nov 2026, selesai 22 Nov 2026.

**Sumber:** skill `imagegen-frontend-web` dipakai lebih dulu, lalu skill **antislop** dipasang lewat `npx skills add miqdadbadjuber/anti-slop` (6 skill ke `.agents/skills/`, ditinjau sebelum dipakai: 15 file baru, tidak ada yang terhapus, `opencode.json` tidak tersentuh). Mode dipilih user: **DURING**, lalu instruksi *"kurangi dosis sesuai batas rule"* untuk tiga benturan yang saya sebutkan lebih dulu.

**Design Read** (diwajibkan sebelum menyentuh UI; DESIGN.md tidak punya baris `Dial:`, jadi diinfer):

> Reading this as: personal finance mobile app for individual users, in a dark glassmorphic fintech style, **dial ENERGY 1 / RHYTHM 1 / MOTION 1**.

Alasan: aplikasi keuangan menuntut ketenangan; §4.7 membatasi motion (150–200ms + hormati `prefers-reduced-motion`); layout bergrid konsisten.

## Masalah

Ketiga aturan yang dilanggar justru dibatasi oleh **DESIGN.md sendiri**, dan implementasi melampauinya:

| Aturan | Batas | Sebelum |
|---|---|---|
| **R-10** glass | maks. 2 permukaan | **6**: kartu, sheet, overlay, navbar, header asisten, bar input |
| **R-13** glow | maks. 1 jenis elemen | **3 jenis / 5 titik**: tombol primer, FAB, sidebar, 2 ikon AI |
| **R-04** ikon | tanpa Sparkle/Star sebagai ikon fitur | **14 lokasi** `Sparkles`, di 7 file |

§4.5 berbunyi *"Use glassmorphism selectively"* — permukaan berblur di 6 tempat sudah melampaui panduan itu sendiri.

## Acceptance criteria

1. Blur turun ke maksimum 2 permukaan; glow hanya pada satu jenis elemen; nol `Sparkles` di seluruh `apps/web`.
2. DESIGN.md §4.5 dan §4.6 disesuaikan **dengan change log** di §1.3 (syarat dokumen `LOCKED`).
3. Token glow yang tidak terpakai dihapus dari `globals.css` agar tidak ada kode mati yang menyesatkan.
4. Teks produk bebas em dash (R-02) setelah keputusan pemilik.
5. Regresi: `lint`, `tsc`, `build`, `test:types`, a11y, e2e tetap hijau.

## Implementasi

**R-10 — blur: 6 → 2 permukaan**

Pertahankan dua yang punya alasan fungsional:
- `.glass-panel` — kartu/panel, yaitu level *Panel* di DESIGN.md §4.5 (permukaan signature).
- Bottom navbar — satu-satunya permukaan tempat konten benar-benar **bergulir di belakangnya**, jadi blur-nya bekerja, bukan dekoratif.

Dihapus: sheet, overlay modal, header asisten, bar input. Keempatnya sudah punya varian solid; tanpa blur, translusen justru membuat teks sulit dibaca (§4.5: *"Do not depend on backdrop-filter for readability"*).

**R-13 — glow: 3 jenis → 1**

Hanya `--shadow-glow-primary` yang tersisa, terpasang pada tombol primer, FAB, dan CTA sidebar (satu jenis: *aksi primer*). `shadow-glow-ai` dilepas dari kedua ikon avatar asisten; token `ai`, `warning`, dan `danger` **dihapus dari `globals.css`** karena tidak dipakai komponen mana pun (`warning`/`danger` memang sudah mati sejak awal).

**R-04 — Sparkles: 14 → 0**

Semua diganti berdasarkan **relevansi dengan isi**, bukan estetika:

| Lokasi | Pengganti | Alasan |
|---|---|---|
| FAB, sidebar, "Buka Asisten", avatar asisten ×2, `navigation.ts` | `MessagesSquare` | Kontrol membuka permukaan percakapan |
| EmptyState "Belum ada transaksi" | `ReceiptText` | Relevan dengan transaksi |
| Badge "Asisten/Tercatat" | **dihapus** | Badge sudah punya teks; ikon tidak menambah informasi |
| Tagline `/login` | **dihapus** | Murni dekoratif, tanpa fungsi |

**R-02 — teks produk**

Empat kalimat diganti (judul halaman, `aria-label` sidebar, pesan galat nama depan, paragraf kartu Sesi), dan 6 placeholder nilai kosong `"—"` → en dash `"–"` atas keputusan pemilik (tampak nyaris sama, tetapi bukan em dash).

## Keputusan pemilik yang dicatat

1. **Mode antislop: DURING** — diterapkan saat menulis, bukan audit sesudahnya.
2. **Kurangi dosis** (bukan pertahankan + override) untuk ketiga benturan.
3. **Placeholder `"—"` → en dash**, karena R-02 dirancang untuk prosa pemasaran.
4. **225 em dash di 31 file dokumentasi dibiarkan** — dianggap di luar cakupan karena R-02 menyasar copy produk (seluruh contohnya kalimat pemasaran), sedangkan ini dokumentasi teknis internal.

## QA

| Pemeriksaan | Hasil |
|---|---|
| `lint` / `tsc` / `build` web | 0 |
| `npm run test:types` | 0 |
| `npm run test:e2e` | **50/50** |
| Lighthouse a11y 8 halaman | **1.0, nol kegagalan** |
| Audit runtime blur (`/home`) | 3 elemen unik → **2 jenis permukaan** |
| Audit runtime glow (`/home`) | 4 elemen, semuanya `bg-primary`/FAB → **1 jenis** |
| Audit runtime (`/assistant`) | `blur: []`, nol `sparkles` |
| Audit kode | `Sparkles` **0** · `backdrop-filter` **hanya** `mobile-bottom-navigation.tsx` · `shadow-glow` **hanya** `shadow-glow-primary` |
| Em dash teks produk | **0** (tersisa 8 komentar kode, di luar cakupan copy) |
| Container | healthy ×3 |

**Dua kegagalan verifikasi saya sendiri, tertangkap dan diperbaiki:**

1. **Script screenshot menampilkan halaman yang salah.** `Page.navigate` async, dan loop membaca `innerText` sebelum navigasi selesai, sehingga ia puas dengan konten halaman sebelumnya. Diperbaiki dengan menunggu `location.pathname` cocok — dan diverifikasi lewat log isi DOM (`p:"/assistant", head:"Asisten FISCA…"`).
2. **`getByText("Liburan uji")` melanggar strict mode** di test savings — notice sukses `Target "Liburan uji" dibuat.` ikut cocok. Dipersempit ke `getByRole("heading")`. Ini terbukti hanya saat suite dijalankan penuh, tidak saat terpisah.

## Delivery Gate (wajib sebelum kirim)

**Block 1 — Hard Gate (semua harus NO)**

| # | Pertanyaan | Status |
|---|---|---|
| R-02 | Em dash di teks? | **NO** — 0 di teks produk; 8 komentar kode dianggap di luar "copy"; 225 dokumentasi atas keputusan pemilik |
| R-03 | Overflow horizontal / layout rusak di mobile? | **NO** — a11y 8/8, suite mobile lulus, CDP 360px (ISSUE-21) |
| R-17 | Statistik tanpa sumber? | **NO** — semua angka dari API/ledger FISCA |
| R-18 | Testimoni fiktif? | **NO** — FISCA tidak punya bagian testimoni |
| R-23 | Aset dibuat tanpa instruksi? | **NO** — tidak ada aset visual baru; ikon dari library, diganti berdasarkan relevansi |
| R-24 | Tautan nav ke tempat tak ada? | **NO** — 8 halaman punya route dan tercakup e2e |
| R-25 | Kontras di bawah WCAG AA? | **NO** — Lighthouse a11y **1.0** di 8 halaman |
| R-26 | Kontrol mati? | **NO** — e2e 50/50 mengklik FAB, tombol, form, nav |
| R-27 | Kurang empty/loading/error state? | **NO** — `EmptyState`/`Skeleton`/`Alert` teruji (mis. savings "Belum ada target") |
| R-28 | FAQ generik? | **NO** — tidak ada FAQ |
| R-32 | Tidak bisa keyboard / tanpa fokus? | **NO** — a11y 1.0, `:focus-visible` global |
| R-33 | Fitur via patch script? | **NO** — semua di sumber |
| R-34 | Satu mode tema rusak? | **NO** — dark-only, tanpa toggle |
| R-35 | Dikirim tanpa dijalankan? | **NO** — build + e2e 50/50 + a11y 8/8 + audit runtime |
| R-36 | Klaim performa/keamanan fiktif? | **NO** — hanya Lighthouse a11y, dan itu benar-benar diukur |
| R-37 | Tanpa arah design? | **NO** — DESIGN.md ada; Design Read dideklarasikan (1/1/1) |
| R-38 | Konten realistis yang dibuat-buat? | **NO** — konten dari API |
| R-39 | Semantik/afordansi kurang? | **NO** — a11y 1.0, ikon `aria-hidden`, tombol punya nama akses |
| R-40 | Media tanpa dimensi / klaim performa tak terukur? | **NO** — tidak ada angka performa yang tidak diukur |

**Block 2 — Purpose-Gate (teknik boleh, alasan wajib tertulis)**

| # | Teknik | Alasan tertulis |
|---|---|---|
| R-01 | Glow/gradien | Hanya pada aksi primer → penanda fokus |
| R-04 | Ikon | Relevansi per fungsi: `MessagesSquare` = kontrol percakapan, `ReceiptText` = transaksi; yang dekoratif dihapus |
| R-09 | Badge | Ikon di badge "Asisten" dihapus karena teks sudah menyampaikan |
| R-10 | Glass | 2 permukaan: kartu (signature) + navbar (konten bergulir di belakang) — alasan di `globals.css` dan DESIGN.md §4.5 |
| R-13 | Glow | 1 jenis: aksi primer — alasan di DESIGN.md §4.6 |

Tidak ada teknik lain yang ditambahkan (R-06, R-07, R-08, R-12, R-14, R-19, R-22 tidak menyentuh issue ini).

**Block 3 — Liveliness**

- Dial **ENERGY 1 / RHYTHM 1 / MOTION 1** dideklarasikan sebelum bekerja dan konsisten: layout FISCA memang grid kolom seragam (RHYTHM 1), tanpa motion dekoratif (MOTION 1), tanpa teriakan visual (ENERGY 1).
- Satu fokus per layar terjaga: Total Saldo + "Catat transaksi" di beranda.

**Block 4 — Craftsmanship & Quality Locks**

| Kunci | Status |
|---|---|
| C-1 Intentionality | ✅ tiap keputusan punya alasan di atas |
| C-2 Functional completeness | ✅ e2e 50/50, tak ada kontrol mati |
| C-3 Content-driven composition | ✅ tidak ada section baru yang ditambah |
| C-4 Resilience | ✅ empty/loading/error state teruji |
| C-5 Evidence over claims | ✅ hanya klaim yang benar-benar diukur |
| R-05/R-11/R-15/R-16/R-20/R-21/R-29/R-30 | ✅ tidak diubah struktur, radius, CTA, palet, atau identitas |
| R-31 Alasan tiap keputusan | ✅ ditulis di issue ini |

**Kesimpulan Gate: PASS (4/4 block).**

## Batasan

- **8 komentar kode** masih memuat em dash. Saya memperlakukannya di luar cakupan R-02 karena aturan itu menyasar copy yang dilihat pengguna; keputusan ini **diungkapkan di sini, bukan disembunyikan**. Bila Anda ingin seluruhnya bersih, tinggal bilang.
- Layout struktural Stitch tetap tidak disamakan (elemen karbon/ESG dilarang plan M7) — keputusan lama di ISSUE-23.
- Tidak ada test otomatis khusus untuk perubahan dosis ini; verifikasi lewat audit runtime, Lighthouse, dan suite regresi. Menambah test yang membedakan "blur ada/tidak" akan rapuh (terikat styling), jadi sengaja tidak dibuat.
