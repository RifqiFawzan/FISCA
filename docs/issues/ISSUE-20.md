# ISSUE-20 — Savings Planner & Purchase Advisor

**Status:** CLOSED (QA PASS)

**Sumber:** PRD §4.2 (Should Have / MVP lanjutan), §10.4 Savings Planning, §10.6 Purchase Affordability, O-17 (savings account model).

**Acceptance criteria**
1. Target tabungan: nominal, tenggat, kontribusi, progres.
2. Purchase Advisor: simulasi kemampuan membeli dengan angka dari database, bukan angka karangan AI.
3. AI tidak boleh mengarang penghasilan/cicilan yang belum dicatat; bila data kurang, nyatakan keterbatasan.
4. Perubahan target mengikuti pola approval yang sama dengan budget bila memengaruhi anggaran.
5. Endpoint baru ditambahkan ke kontrak API dan tercakup test integrasi.

**Implementasi**
- `db/migrations/0002_savings.sql`: tabel `savings_goals`. Sebuah goal adalah **target, bukan dompet**: `account_id` menautkannya ke akun yang sudah ada (PRD O-17), `version` untuk optimisme, `archived_at` untuk arsip — **tidak ada kolom saldo sama sekali**, jadi uang tidak pernah bisa dihitung dua kali.
- `db/queries/savings.sql` + `CountActiveMonths` (untuk jujur soal kedalaman riwayat).
- `internal/service/savings.go`: `Create/List/Get/Update/ArchiveSavingsGoal` (progres, sisa, persen, status, bulan tersisa, kebutuhan/bulan, proyeksi) dan `PurchaseAdvisor` (saldo ledger − kekurangan target = dana bebas; rata-rata masuk/keluar 90 hari; asumsi + keterbatasan).
- 6 endpoint baru: `GET/POST /savings-goals`, `GET/PATCH/DELETE /savings-goals/{id}`, `GET /purchase-advisor?price=`.
- Kontrak `docs/api/openapi.yaml`: +3 path (31 total), 5 skema baru, `AuditEntry.action` + `archive`; `npm run gen:api` dijalankan ulang.
- Web: `components/views/savings-view.tsx`, rute `/savings`, entri **Tabungan** di sidebar desktop (nav bawah mobile tetap 5 item sesuai DESIGN §7.1), kartu Tabungan di `/home` agar bisa dijangkau dari mobile (PRD §7 "halaman turunan dari Home").

**QA**
- Unit `internal/service/savings_status_test.go`: 6 kasus status (termasuk grace period) + 6 kasus `monthsUntil` — lulus.
- Integrasi `internal/server/savings_test.go`: progres dari ledger + **tanpa double counting** (membuat goal dan memindahkan uang antar akun tidak mengubah `available`), simulasi `?monthly=` (tercapai & tidak), versi basi → 409, arsip → 404, isolasi lintas-user → 404 (7 kasus validasi input juga), `PurchaseAdvisor` (angka ledger + `reservedSavings` + keterbatasan) — lulus.
- Kontrak: fungsi cakupan **menangkap 6 endpoint baru** yang belum saya panggil; setelah ditambahkan, 31 path lulus.
- `go test ./... -count=1` + `go vet` bersih; `tsc`/`lint`/`build` bersih (rute `/savings` termuat); `stack:up` + `smoke` lulus; `test:e2e` **24/24 lulus**.
- Browser: buat target via form (5jt → tampil progres + "butuh Rp2.500.000/bulan"), advisor menampilkan asumsi + keterbatasan, sidebar memuat "Tabungan"; CDP 360px `scrollWidth=360` di 8 halaman.

**Temuan QA selama pengerjaan (semua diperbaiki)**
1. Goose gagal mem-parse migration tanpa anotasi `-- +goose Up` / `-- +goose Down`.
2. `monthsToTarget = -1` tidak mungkin tercapai lewat API (`monthly` selalu > 0) → sentinel dihapus, deskripsi kontrak dan test disesuaikan.
3. Goal baru langsung berstatus **Perlu dipantau** karena `elapsed > 0` walau baru beberapa milidetik → ditambah grace period 10% jendela + unit test.
4. `new Date()` di dalam sheet gagal prerender Next 16 (`cacheComponents`) → sheet di-mount hanya saat dibuka, sehingga default tenggat dihitung di browser.
5. `Badge` tidak punya tone `success`/`error` → dipetakan ke `income`/`danger`.

**Batasan**
- Progres dihitung dari **saldo akun yang ditautkan**; bila akun itu juga dipakai untuk keperluan lain, angka progres ikut terpengaruh. Kontrak menyatakan hal ini.
- PRD §10.4 menyebut "kontribusi per periode": di sini di**hitung** (`requiredMonthly`, proyeksi), bukan disimpan sebagai baris transaksi — belum ada buku kas tabungan terpisah.
- Menghapus tautan akun tidak didukung (hanya menautkan ulang); target tidak bisa berdiri tanpa akun.
- ~~Belum ada test E2E Playwright khusus halaman ini~~ **Ditutup oleh ISSUE-25 (13 Nov 2026)**: `tests/e2e/savings.spec.ts` menguji pembuatan target (tampil di layar **dan** tersimpan di server) serta advisor yang membalas dari data nyata. Contract test dan integrasi test tetap berlaku.
- Daftar endpoint di PRD §17 belum memuat endpoint savings; penambahannya dicatat di `plan.md` §7.2.
