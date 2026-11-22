# ISSUE-23 — Penyelarasan design Dark Carbon Glass: warna, badge, distribusi, tile

**Status:** CLOSED (QA PASS) — dikerjakan 10 Nov 2026, dicatat 11 Nov 2026.

**Sumber:** permintaan user menyelaraskan UI dengan design Stitch project `2264071121943649738` ("Fisca Application Project"), arah **Dark Carbon Glass**. Pengerjaannya selesai lebih dulu tetapi **baru dicatat di sini** — sebelumnya hanya 5 deviasi warna yang tercatat di `plan.md` §17.2, sementara perubahan komponennya tidak punya jejak di dokumen issue mana pun.

**Referensi visual:** `design-refs/` — 10 screen Dark Carbon Glass, ditarik lewat REST dengan header `X-Goog-Api-Key` (`scripts/pull-stitch-screens.cjs`) lalu disaring per aset design system agar varian lain tidak tercampur (`scripts/unwrap-stitch-screenshots.cjs` untuk membuka wrapper JSON-nya).

**Acceptance criteria**
1. Warna UI mengikuti palet Dark Carbon Glass **tanpa** mengorbankan WCAG AA (4,5:1).
2. Status budget tampil tiga tahap, dan warna bar progres selalu sama dengan badge-nya.
3. Distribusi pengeluaran di beranda tampil sebagai bar bertumpuk dengan pill persentase.
4. Baris transaksi memakai tile ikon berwarna kategori dan menampilkan label sumber.
5. `lint`, `tsc`, `build`, Lighthouse a11y 8 halaman, dan e2e tetap hijau.

**Implementasi**

**1. Warna — 5 deviasi (rincian di `plan.md` §17.2)**

| Token | Lama | Baru | Pendorong |
|---|---|---|---|
| `muted-dim` | `#64748B` | `#8493AB` | WCAG: 3,0–4,1:1 di semua permukaan |
| `danger` | `#F43F5E` | `#F66981` | WCAG: 4,16:1 di chip badge ber-tint |
| `primary` | `#10B981` | `#4EDEA3` | Palet Stitch (`primary-fixed-dim`) |
| `primary-hover` | `#059669` | `#46C893` | Palet Stitch |
| `primary-deep` | `#047857` | `#3EB282` | Palet Stitch |
| `ai` | `#06B6D4` | `#4CD7F6` | Palet Stitch (`secondary`) |
| `analytics` | `#6366F1` | `#C0C1FF` | Palet Stitch (`tertiary`) |

Yang perlu diingat: **dua deviasi pertama lahir dari kegagalan WCAG, bukan selera** (keduanya tercatat sejak ISSUE-21). Lima sisanya mengikuti palet Stitch, dan rasio kontrasnya justru **lebih baik** daripada nilai lama (teks 7,20 vs 4,84). Yang menarik: `primary-hover` dan `primary-deep` lama **gagal** WCAG (3,50 dan 2,41) dan tidak pernah tertangkap Lighthouse karena hanya mengaudit keadaan diam; kini 6,25 dan 4,96. Glow `--shadow-glow-primary` dan `::selection` ikut disamakan ke mint.

**2. Badge tiga tone** — `components/ui/progress.tsx`

Stitch menampilkan status sebagai `78% Normal`, `92% Mendekati Limit`, `110% Over Limit`. Sebelumnya hanya dua tone dan logikanya ditulis inline terduplikasi di `budget-view`.

- `statusForPercent(percent)` → `{ tone, label }`: `≥1` → `danger`/"Over Limit", `≥0.8` → `info`/"Mendekati Limit", selain itu → `primary`/"Normal".
- `toneForPercent()` kini mendelegasikan ke helper yang sama, sehingga **bar dan badge mustahil berbeda warna**.
- `Badge` mendapat tone `info` (`bg-ai/15 text-ai`); `Progress` mengenal `info` → `bg-ai`.
- `budget-view`: badge header memakai helper, plus **badge baru per kategori** seperti Stitch.

**3. `DistributionBars` di beranda** — `components/views/home-view.tsx`

Menggantikan `<ul>` berisi `Progress` per kategori. Sesuai yang ditegaskan user: **kartu Total Saldo (hero) tidak diganti** — distribusi adalah kartu terpisah.

- Satu bar bertumpuk per kategori + pill legenda (dot warna · nama · persentase) + pill **"Sisa"** bila sisanya > 0.
- Lebar segmen dinormalisasi, karena API bisa mengembalikan share > 1 atau < 1 — bar tetap penuh.
- Bar diberi `aria-hidden`: seluruh nilainya sudah tersedia sebagai teks di pill, sehingga pembaca layar tidak membacanya dua kali.
- Sumber data `CategorySpend` (`name`, `color`, `total`, `percent`) sudah tersedia — **tanpa perubahan backend**.

**4. Tile baris transaksi** — `components/finance/transaction-row.tsx`

| Aspek | Lama | Baru |
|---|---|---|
| Bentuk | `rounded-full ring-1`, `size-10` | **`rounded-lg`, `size-12`** (48px, radius 16px) |
| Warna | Berdasar jenis transaksi | **Warna kategori** (`categoryColor` + alpha 15%), fallback jenis |
| Ikon | 16px | 20px |
| Label sumber | tidak ada | **badge `Voice`/`Chat`** |

Dua hal di sini menutup requirement, bukan sekadar meniru Stitch:
- **DESIGN.md §8.3 mensyaratkan** *"Optional source label: Voice, Chat, or Manual"* pada tiap item transaksi — itu belum terpenuhi. Badge hanya untuk `voice`/`chat`; `manual` adalah default dan DESIGN.md menyebutnya *optional*, jadi tidak ditampilkan agar baris tidak bising. (`Bank Sync`/`QRIS` tidak diikutkan: yang pertama butuh integrasi bank, dan DESIGN.md tidak mensyaratkannya.)
- **Jebakan radius yang saya temukan sendiri:** percobaan pertama memakai `rounded-xl`, tetapi `--radius-xl` ditimpa menjadi **24px** oleh DESIGN.md §4.4 — di kotak 44px itu melebihi setengah sisi sehingga browser membulatkannya jadi lingkaran penuh. Tertangkap lewat `getComputedStyle` (radius 24px), bukan dari nama kelasnya.

**Keputusan user saat pengerjaan**
- **FAB tetap `Sparkles`** — Stitch memakai mikrofon, tapi mengubahnya berarti mengubah fungsi (chat → rekam suara), bukan sekadar ikon.
- **Hero Total Saldo tetap** — distribusi adalah kartu terpisah di bawah.

**QA**
- Kontras dihitung secara deterministik sebelum menerapkan (script lokal), bukan ditebak.
- Lighthouse a11y **8 halaman = 1.0** setelah setiap perubahan terakhir (yang paling akhir setelah tile transaksi).
- e2e saat pengerjaan 24/24; suite kini **34/34** setelah ISSUE-15 dan ISSUE-22.
- Pemeriksaan DOM: tile 48×48 dengan radius 16px; pill menampilkan `Uncategorized 87%` / `Belanja 13%` dan pill "Sisa" tidak muncul karena sudah 100%.

**Di luar cakupan**
- **Layout struktural Stitch tidak disamakan.** Header lonceng, chip "Low-Carbon Mode", "SYNC AKTIF", dan tab "Kelola Budget / Laporan Finansial" tidak diimplementasikan — sebagian adalah elemen karbon/ESG yang **dilarang plan M7**. Yang disamakan hanya bahasa visual (warna, badge, tile, distribusi), karena itu domain DESIGN.md.
- Data uji hanya 2 kategori, sehingga bar bertumpuk belum pernah terlihat dengan 4 segmen seperti di Stitch.
- **Test otomatis:** threshold tiga tone kini tercakup test e2e **"budget badge renders all three tones"** di `tests/e2e/budget.spec.ts` (3 kasus × 2 viewport) — memverifikasi teks badge sekaligus kecocokannya dengan persentase dari API. `DistributionBars` **masih belum** ber-test langsung; verifikasinya lewat Lighthouse, e2e regresi, dan pemeriksaan DOM.
