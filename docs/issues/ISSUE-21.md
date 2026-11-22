# ISSUE-21 — Audit a11y & responsivitas UI

**Status:** CLOSED (QA PASS)

**Sumber:** DESIGN.md §105 ("readable text, sufficient contrast, visible focus states, clear labels, touch targets suitable for mobile use"), PRD §22.2, permintaan perbaikan design UI.

**Acceptance criteria**
1. Tiap halaman dirender di 360px dan 1280px, lalu diaudit untuk a11y dan responsivitas.
2. Temuan kontras/ARIA/touch target diperbaiki tanpa menyimpang sembarangan dari DESIGN.md.
3. Perubahan terhadap token DESIGN.md dicatat sebagai deviasi, bukan disunting diam-diam.
4. Tidak ada regresi: lint, build, E2E, dan smoke tetap hijau.

**Implementasi & artefak**
- `design-refs/` (19 screen Stitch, 5,5 MB) ditarik lewat REST `X-Goog-Api-Key` karena `sp_screens` MCP masih gagal (bug plan.md §15) — script: `scripts/pull-stitch-screens.cjs`. **Folder ini sudah dihapus** (isinya terlalu bercampur antar varian); tarik ulang bila diperlukan, lalu jalankan `scripts/unwrap-stitch-screenshots.cjs`.
- `audit-html/` — 16 file HTML hasil render 8 halaman × 2 lebar; script: `scripts/capture-audit-html.cjs`.
- Perbaikan: `--color-muted-dim` `#64748B → #8493AB`, `--color-danger` `#F43F5E → #F66981`, `/login` dibungkus `<main>`.

**QA**
- Lighthouse (axe) a11y 8 halaman @1280: **semua 1.0, nol kegagalan** (sebelumnya 6 halaman 0,96 karena `color-contrast` dan `/login` 0,93 karena `landmark-one-main`).
- Stitch `sp_a11y` (login & savings): `warnings: []` — termasuk pemeriksaan touch target yang tidak dicakup Lighthouse.
- Stitch `sp_analyze`: `hasViewport`, `hasBreakpoints`, `hasAccessibility`, `hasTailwind` = true.
- CDP 360px 8 halaman: `scrollWidth=360`, nol elemen keluar viewport.
- `npm run lint`, `npm run build`, `npm run test:e2e` **24/24**, `npm run smoke:api` — semua hijau.

**Temuan QA**
1. Konflik DESIGN.md §4 dengan §105: `#64748B` dan `#F43F5E` tidak memenuhi WCAG AA di permukaan gelap. Dipilih nilai terdekat yang lulus (`#8493AB`, `#F66981`) sehingga peran dan hierarki warna tetap sama; dicatat di `plan.md` §17.2.
2. HTML audit awal berupa satu baris panjang sehingga terpotong saat dibaca → script sekarang merapikan per baris.
3. `sp_analyze` menyarankan `sp_extract`/`sp_to_react` (konversi ke shadcn/React) — tidak relevan: kita sudah React dan sengaja tidak memakai shadcn (plan.md §2 DECIDED).

**Batasan:** audit Stitch hanya dijalankan di 2 halaman (login, savings) sebagai cross-check karena menerima HTML inline; 8 halaman dilindungi Lighthouse + CDP. `sp_screens` tetap gagal sehingga perbandingan pixel-per-pixel dengan screen Stitch asli belum dilakukan.
