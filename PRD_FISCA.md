# AI PERSONAL FINANCE — FINAL PRODUCT REQUIREMENTS DOCUMENT

**Nama kerja produk:** AI Personal Finance  
**Versi:** 1.2 — Consolidated Product, UI/UX, and Technical Baseline  
**Tanggal:** 8 November 2026  
**Bahasa dokumen:** Indonesia  
**Platform:** Responsive Web, Mobile-first  
**Status:** Baseline final untuk memulai technical design dan implementasi bertahap.  
**Product owner:** Pengguna / pemilik produk  

> **Catatan status:** Dokumen ini menggabungkan keputusan brainstorming produk, UX, dan teknologi. `LOCKED` berarti dipilih secara eksplisit oleh product owner. `PROPOSED` berarti rekomendasi yang belum disetujui secara eksplisit. `OPEN` berarti masih membutuhkan keputusan atau validasi. Jangan mengubah status tanpa keputusan product owner.

---

# 0. Ringkasan Eksekutif

AI Personal Finance adalah aplikasi web pencatat dan pengelola keuangan pribadi yang memungkinkan pengguna mencatat pemasukan, pengeluaran, dan transfer melalui bahasa alami menggunakan suara maupun teks. Aplikasi mengelompokkan transaksi, menyajikan dashboard dan laporan, memungkinkan pencarian riwayat transaksi dengan pertanyaan alami, serta menyediakan analisis dan rekomendasi keuangan berbasis data aktual pengguna.

Produk dimulai sebagai **personal MVP**, tetapi arsitektur harus mendukung **SaaS multi-user** sejak awal. Pengalaman mobile menjadi prioritas desain walaupun aplikasi tetap mendukung tablet dan desktop.

## 0.1 Keputusan Inti yang Sudah Dikunci

| Area | Keputusan | Status |
|---|---|---|
| Target produk | Personal MVP dengan fondasi SaaS multi-user | LOCKED |
| Platform | Responsive web | LOCKED |
| Pendekatan UX | Mobile-first | LOCKED |
| Interaksi AI | Voice dan Chat | LOCKED |
| Pencatatan transaksi | Hybrid Recording | LOCKED |
| Primary action mobile | Tombol AI di tengah Bottom Navigation | LOCKED |
| Perilaku tombol AI | Smart Assistant: membuka assistant dan siap menerima voice; pengguna dapat beralih ke chat | LOCKED |
| Assistant mobile | Full-screen; Bottom Navigation disembunyikan selama Assistant aktif | LOCKED |
| Pencarian transaksi | AI dapat menjawab kapan transaksi terjadi dan berapa nominalnya | LOCKED |
| Analisis transaksi besar | Largest Transaction saat ditanyakan | LOCKED |
| Analisis pengeluaran tidak biasa | Unusual Spending saat ditanyakan | LOCKED |
| Peringatan proaktif pengeluaran | Tidak termasuk MVP | LOCKED |
| Budget structure | Combined Budget: Overall Budget + Category Budget | LOCKED |
| Pembuatan budget | Hybrid Budget: manual atau rekomendasi AI | LOCKED |
| Pengelolaan budget dengan AI | Conversational Management melalui voice/chat | LOCKED |
| Persetujuan perubahan budget | Wajib user approval sebelum perubahan diterapkan | LOCKED |
| Budget terlampaui | Warning Only; transaksi tetap boleh dicatat | LOCKED |
| Frontend framework | Next.js | LOCKED |
| Backend language | Go / Golang | LOCKED |
| Backend HTTP framework | Gin | LOCKED |
| Main database | PostgreSQL | LOCKED |
| Container environment | Gunakan Docker yang sudah tersedia pada komputer pengguna; Docker Compose untuk mengelola service lokal | LOCKED |

## 0.2 Keputusan yang Belum Dikunci

| Area | Rekomendasi / pertanyaan | Status |
|---|---|---|
| Brand dan nama produk | Nama kerja masih AI Personal Finance | OPEN |
| Visual identity | Modern minimalist/fintech sebagai arah awal | OPEN |
| Warna final | Aksen hijau diusulkan, belum disetujui | OPEN |
| Tema | Light, dark, atau keduanya | OPEN |
| Typography | Inter atau Plus Jakarta Sans sebagai kandidat | OPEN |
| Account model | Hybrid Accounts (Cash default + akun bank/e-wallet) direkomendasikan | OPEN |
| Auth provider | Supabase Auth atau solusi setara | PROPOSED |
| Database access | pgx + sqlc direkomendasikan untuk Go | PROPOSED |
| AI provider | OpenAI API dan Official Go SDK direkomendasikan | PROPOSED |
| Speech-to-Text provider | OpenAI Transcription atau provider setara; wajib uji Bahasa Indonesia | PROPOSED |
| Realtime voice | Evaluasi setelah flow voice dasar stabil | OPEN / FUTURE |
| Budget period | Bulanan sebagai default MVP | PROPOSED |
| Budget rollover | Sisa budget tidak otomatis ditambahkan ke bulan berikutnya | PROPOSED |
| Warning threshold | 80% sebagai peringatan awal | PROPOSED |
| Aturan unusual-spending | Baseline statistik, minimum histori, dan threshold | OPEN |
| Refund, utang, pinjaman | Definisi domain sebelum implementasi fitur terkait | OPEN |

---

# BAGIAN I — PRODUCT REQUIREMENTS

# 1. Product Vision

## 1.1 Visi

Membuat pengelolaan keuangan pribadi menjadi lebih mudah, cepat, dan natural. Pengguna dapat mencatat, mencari, memahami, dan merencanakan keuangan melalui percakapan dengan AI tanpa harus mengisi banyak form secara manual.

## 1.2 Masalah Pengguna

1. Pencatatan manual terasa merepotkan sehingga transaksi kecil sering terlupakan.
2. Pengguna kesulitan mengetahui kategori yang paling banyak menghabiskan uang.
3. Riwayat transaksi sulit dicari ketika pengguna lupa tanggal atau nominal.
4. Laporan angka saja belum cukup membantu pengguna memahami kondisi finansial.
5. Pengguna membutuhkan bantuan menyusun budget dan target tabungan, tetapi tetap ingin memegang kendali.

## 1.3 Solusi Produk

Satu aplikasi personal finance dengan:

- Transaction capture melalui voice, chat, dan form manual.
- Kategorisasi transaksi dan pengelolaan akun/dompet.
- Dashboard, riwayat, dan laporan.
- AI Smart Assistant untuk pencatatan dan pertanyaan keuangan.
- Historical Transaction Intelligence.
- Largest Transaction dan Unusual Spending saat diminta.
- Financial Analyst dan Financial Advisor.
- Combined Budget dengan conversational management dan approval.

## 1.4 Prinsip Produk

1. **Voice-first, chat-enabled:** voice menjadi primary action di mobile, tetapi chat tetap menjadi alternatif penuh.
2. **Financial accuracy first:** nominal, saldo, dan agregasi dihitung backend/database; LLM tidak boleh mengarang angka.
3. **User remains in control:** AI boleh menganalisis dan mengusulkan, tetapi perubahan budget memerlukan approval.
4. **Low-friction capture:** transaksi jelas dapat dicatat otomatis; transaksi ambigu meminta klarifikasi.
5. **Privacy by default:** setiap pengguna hanya dapat mengakses data miliknya sendiri.
6. **Mobile-first, responsive everywhere:** mobile dirancang khusus, bukan sekadar desktop yang diperkecil.
7. **Explainable insights:** insight menyertakan dasar data, periode, dan keterbatasan yang relevan.
8. **One assistant experience:** kemampuan AI berada dalam satu Smart Assistant, bukan chatbot terpisah untuk setiap modul.

# 2. Tujuan, Non-Goals, dan Success Metrics

## 2.1 Tujuan

- Mempercepat pencatatan pemasukan, pengeluaran, dan transfer.
- Mengurangi kesalahan, transaksi terlupa, serta input ganda.
- Memudahkan pencarian transaksi historis dengan bahasa alami.
- Menampilkan kondisi keuangan secara mudah dipahami.
- Membantu pengguna membuat dan mengevaluasi budget.
- Memberikan rekomendasi yang didukung data aktual.
- Menyediakan fondasi aman untuk SaaS multi-user.

## 2.2 Non-Goals MVP

- Integrasi rekening bank otomatis/open banking.
- Otomatisasi pembayaran atau transfer uang sungguhan.
- Investment trading atau rekomendasi investasi profesional.
- Debt management kompleks dan loan servicing.
- Prediksi keuangan jangka panjang yang mengklaim kepastian.
- Proactive unusual-spending alerts.
- Native Android/iOS application.
- OCR struk sebagai alur inti.
- Microservices atau Kubernetes sejak awal.

## 2.3 Success Metrics (Target Ditetapkan Setelah Pilot)

| KPI | Definisi |
|---|---|
| Transaction capture success | Persentase perintah pencatatan yang menghasilkan transaksi valid |
| Parsing correction rate | Persentase transaksi AI yang perlu dikoreksi pengguna |
| Duplicate transaction rate | Persentase perintah yang menyebabkan duplikasi |
| Voice task completion | Persentase tugas voice yang berhasil diselesaikan |
| Time to record | Waktu dari input dimulai hingga transaksi tersimpan |
| Financial answer accuracy | Kesesuaian tanggal/nominal AI dengan hasil backend |
| Budget proposal approval rate | Persentase proposal yang disetujui |
| Weekly active users | Pengguna aktif mingguan |
| Retention | Pengguna yang kembali menggunakan aplikasi dalam periode tertentu |

Target numerik KPI belum ditetapkan. Nilai harus ditentukan melalui usability test, pilot, dan evaluasi data.

# 3. Target Pengguna dan Platform

## 3.1 Target Pengguna

MVP ditujukan untuk individu yang ingin mengelola keuangan pribadi, mencatat transaksi secara cepat, mencari riwayat, memahami pengeluaran, dan mengelola budget melalui percakapan.

## 3.2 Multi-user Readiness

Walaupun penggunaan awal dapat berfokus pada satu pemilik produk, struktur data dan otorisasi harus siap untuk banyak pengguna:

- Setiap data finansial memiliki owner/user scope.
- Data antar-user tidak boleh saling terlihat.
- Query backend menggunakan identitas dari session/token yang tervalidasi.
- AI tidak menentukan `user_id` sendiri.
- Pengujian harus mencakup percobaan akses lintas-user.

## 3.3 Platform

**LOCKED:** responsive web, mobile-first.

| Platform | Pengalaman |
|---|---|
| Mobile browser | Prioritas desain; bottom navigation, full-screen Assistant, kontrol ramah sentuhan |
| Tablet | Layout adaptif dan penggunaan ruang yang efisien |
| Desktop browser | Sidebar, dashboard multi-kolom, tabel dan panel analitik yang lebih luas |

PWA dan native apps belum termasuk scope MVP.

# 4. Scope dan Prioritas MVP

## 4.1 Must Have

- Authentication dan user profile.
- Multi-user data isolation.
- Account/wallet dasar (detail model perlu keputusan final).
- Income, expense, transfer.
- Category dan subcategory.
- Manual transaction entry.
- Voice dan chat transaction capture.
- Hybrid Recording.
- Transaction history, search, filter, edit, void/Undo.
- Dashboard dasar.
- Reports dasar.
- Smart Assistant dengan Voice + Chat.
- Historical Transaction Intelligence.
- Largest Transaction.
- Unusual Spending saat ditanyakan.
- Financial Q&A berdasarkan query/perhitungan backend.
- Combined Budget.
- Conversational Budget Management dengan approval.
- Warning Only saat budget terlampaui.
- Audit log untuk perubahan penting.

## 4.2 Should Have / MVP Lanjutan

- Savings Goals dan Savings Planner.
- Spending Optimization lebih mendalam.
- Purchase Affordability Analysis.
- Perbandingan periode lanjutan.
- Budget insight yang lebih kaya.

## 4.3 Future

- Realtime voice conversation.
- Background jobs untuk laporan terjadwal.
- Integrasi bank/open banking.
- Receipt OCR.
- Forecasting lanjutan.
- Debt repayment planning.
- Native mobile application.

# 5. Information Architecture dan Navigation

## 5.1 Mobile Bottom Navigation

**LOCKED:** lima posisi navigasi, dengan AI Assistant di tengah sebagai primary action.

| Urutan | Menu | Fungsi |
|---|---|---|
| 1 | Home | Dashboard dan ringkasan keuangan |
| 2 | History | Riwayat transaksi |
| 3 | Assistant | Aksi utama Voice + Chat |
| 4 | Reports | Laporan dan analitik |
| 5 | Profile | Pengaturan, account, kategori, dan privasi |

Assistant button harus lebih menonjol dibanding item navigasi lain, misalnya circular/elevated button. Bentuk, warna, dan motion final masih mengikuti visual identity yang akan diputuskan.

## 5.2 Smart Assistant Behavior

**LOCKED:** pilihan C — Smart Assistant.

Saat tombol Assistant ditekan:

- Buka halaman Assistant.
- Jika microphone permission sudah tersedia, halaman siap menerima voice input.
- Pengguna dapat mulai berbicara atau beralih ke Chat.
- Jika permission belum tersedia, jelaskan kebutuhan izin dan tetap sediakan Chat.

## 5.3 Full-screen Assistant

**LOCKED:** pilihan A — Full-screen Assistant.

- Assistant memakai seluruh area layar pada mobile.
- Bottom Navigation disembunyikan selama Assistant aktif.
- Header menyediakan Back navigation.
- Voice dan Chat berada dalam satu conversation experience.
- Riwayat percakapan dapat dilanjutkan sesuai kebijakan penyimpanan percakapan.
- Saat kembali ke halaman sebelumnya, navigasi utama muncul kembali.
- Jika pengguna keluar ketika recording berjalan, minta konfirmasi untuk membatalkan recording atau tetap berada di Assistant.
- Hasil transaksi yang sudah tersimpan tidak boleh hilang hanya karena pengguna keluar dari Assistant.

## 5.4 Sitemap

```text
Application
├── Authentication
│   ├── Login
│   ├── Register
│   └── Password Recovery
│
├── Home
│   ├── Financial Summary
│   ├── Cash Flow
│   ├── Expense Breakdown
│   ├── Recent Transactions
│   └── Budget Snapshot
│
├── History
│   ├── Transaction List
│   ├── Search & Filters
│   └── Transaction Detail / Edit
│
├── Smart Assistant (Full-screen on mobile)
│   ├── Voice Interaction
│   ├── Chat Interaction
│   ├── Conversation History
│   ├── Transaction Result Cards
│   ├── Analysis Cards
│   └── Clarification Cards
│
├── Reports
│   ├── Income vs Expense
│   ├── Net Cash Flow
│   ├── Category Breakdown
│   └── Monthly Trend
│
└── Profile
    ├── Personal Information
    ├── Accounts / Wallets
    ├── Categories
    ├── Budget Settings
    ├── AI Preferences
    ├── Notifications
    └── Security & Privacy
```

Budget dan Savings Goals dapat menjadi halaman turunan dari Home/Reports; posisi final dalam navigasi ditentukan saat prototyping.

---

# BAGIAN II — FINANCIAL CORE REQUIREMENTS

# 6. Transaction Management

## 6.1 Jenis Transaksi

| Jenis | Definisi | Dampak saldo akun |
|---|---|---|
| Income | Dana diterima | Menambah akun tujuan |
| Expense | Dana dibelanjakan/dibayar | Mengurangi akun sumber |
| Transfer | Pemindahan dana antar-account milik pengguna | Mengurangi akun asal dan menambah akun tujuan; tidak mengubah total dana |

Transfer tidak boleh dihitung sebagai income atau expense dalam laporan arus kas.

## 6.2 Atribut Transaksi

| Field | Tujuan | Aturan |
|---|---|---|
| `id` | Identitas transaksi | UUID |
| `user_id` | Pemilik transaksi | Wajib, diambil dari identity terverifikasi |
| `type` | Jenis transaksi | income / expense / transfer |
| `amount` | Nominal | Bilangan positif; IDR disimpan sebagai integer rupiah |
| `currency` | Mata uang | Default IDR |
| `description` | Deskripsi | Wajib untuk pengalaman pengguna; dapat diisi dari intent |
| `category_id` | Kategori | Nullable jika belum diketahui; transfer tidak memakai kategori expense |
| `account_id` | Akun | Akun untuk income/expense sesuai aturan domain |
| `from_account_id` | Akun asal | Khusus transfer |
| `to_account_id` | Akun tujuan | Khusus transfer |
| `payment_method` | Cara pembayaran | Nullable; contoh cash, QRIS, debit, transfer |
| `merchant_name` | Nama merchant | Nullable |
| `occurred_at` | Waktu transaksi terjadi | Berbeda dari created_at; memakai timezone yang konsisten |
| `notes` | Catatan | Opsional |
| `source` | Asal input | manual / voice / chat |
| `status` | Status transaksi | active / voided atau status setara |
| `created_at` | Waktu pencatatan | Timestamp |
| `updated_at` | Waktu perubahan | Timestamp |

**Aturan nominal:** untuk IDR, simpan angka bulat dalam satuan rupiah menggunakan tipe integer yang cukup besar, misalnya BIGINT. Hindari floating-point untuk nominal uang. Jika multi-currency ditambahkan, aturan presisi dan minor unit harus ditetapkan per mata uang.

## 6.3 Kategori Default yang Diusulkan

**Expense categories:**
- Food & Beverage: makanan, minuman, kopi, restoran.
- Transportation: bensin, parkir, transportasi umum.
- Shopping: pakaian, kebutuhan pribadi, elektronik.
- Bills & Utilities: listrik, air, internet, pulsa.
- Housing: sewa, perawatan tempat tinggal.
- Education: kursus, buku, biaya pendidikan.
- Healthcare: obat, dokter, pemeriksaan.
- Entertainment: hiburan, game, streaming.
- Other / Uncategorized.

**Income categories:**
- Salary.
- Freelance.
- Business.
- Investment Returns.
- Gifts & Allowances.
- Other Income.

Kategori default merupakan rekomendasi awal. Pengguna dapat membuat kategori sendiri. Kategori user-created harus memiliki owner scope.

## 6.4 Account vs Payment Method

Account dan payment method adalah dua konsep berbeda:

- Account: tempat dana berada atau sumber/tujuan dana, misalnya Cash, BCA, Mandiri, GoPay.
- Payment method: cara pembayaran, misalnya tunai, QRIS, kartu debit, atau transfer.

Contoh: "Beli kopi Rp20.000 pakai QRIS dari BCA" berarti account = BCA dan payment_method = QRIS.

**OPEN:** Model account belum diputuskan. Opsi:
1. Single Wallet.
2. Multiple Accounts.
3. Hybrid Accounts: Cash default dan pengguna dapat menambahkan bank/e-wallet.

Hybrid Accounts direkomendasikan tetapi belum LOCKED.

## 6.5 Edit, Undo, Refund, dan Debt

- Edit transaksi harus memperbarui hasil perhitungan dan mencatat audit log.
- Undo sebaiknya menggunakan status voided/cancelled atau mekanisme reversal yang dapat diaudit, bukan hard delete tanpa histori.
- Refund idealnya ditautkan ke transaksi asal sehingga laporan gross dan net expense konsisten.
- Pembayaran utang, penerimaan pinjaman, dan pengeluaran terkait utang memerlukan definisi domain tersendiri.
- Refund dan utang belum menjadi fitur inti MVP sebelum aturan akuntansi produk disetujui.

# 7. Hybrid Recording — Voice & Chat Capture

**LOCKED:** Hybrid Recording.

## 7.1 Aturan

1. Transaksi yang intent, jenis, dan nominalnya jelas dapat disimpan otomatis.
2. Informasi ambigu atau informasi penting yang belum tersedia meminta klarifikasi.
3. Jika hanya kategori yang tidak diketahui, transaksi dapat memakai Uncategorized atau fallback yang jelas.
4. Account yang tidak diketahui mengikuti kebijakan default account setelah diputuskan; jangan mengarang account.
5. Pengguna dapat mengedit atau Undo transaksi.
6. Voice dan Chat menggunakan pipeline parsing, validasi, dan business logic yang sama.
7. Pertanyaan keuangan tidak boleh salah dianggap sebagai perintah pencatatan.
8. Sistem hanya menyatakan sukses setelah backend mengonfirmasi penyimpanan.

## 7.2 Pipeline

```text
Voice / Chat / Manual Input
          |
          v
Speech-to-Text (untuk Voice)
          |
          v
Intent Recognition
          |
          v
Entity Extraction
(type, amount, date, category, account, merchant, payment method)
          |
          v
Backend Validation
          |
     +----+----+
     |         |
  Valid     Ambiguous
     |         |
 Save       Draft + Clarification
     |         |
     +----+----+
          |
          v
Transaction Result / User Feedback
```

## 7.3 Contoh Perintah

| Perintah | Ekspektasi |
|---|---|
| "Saya beli makan 25 ribu" | Expense Rp25.000; kategori makanan atau fallback |
| "Dapat freelance 800 ribu" | Income Rp800.000 |
| "Beli makan 25 ribu dan kopi 18 ribu" | Dua transaksi terpisah |
| "Kemarin beli bensin 50 ribu" | Expense dengan occurred_at kemarin |
| "Pindahkan 300 ribu dari BCA ke Cash" | Transfer antar-account |
| "Ubah transaksi kopi tadi jadi 15 ribu" | Identifikasi transaksi kontekstual, lalu validasi mutasi |
| "Hapus transaksi makan tadi" | Identifikasi target tepat, tampilkan/konfirmasi bila ambigu, lalu void |
| "Berapa pengeluaran saya minggu ini?" | Pertanyaan analitik, bukan transaksi |

## 7.4 Clarification

Contoh:

- User: "Saya tadi bayar 50 ribu."
- AI: "Pembayaran Rp50.000 itu untuk apa?"

Draft yang menunggu klarifikasi tidak boleh memengaruhi saldo, dashboard, laporan, atau budget.

## 7.5 Idempotency dan Retry

- Perintah yang sama terkirim ulang karena retry jaringan tidak boleh membuat transaksi ganda.
- Gunakan idempotency key atau mekanisme deduplication yang tervalidasi.
- Selama processing, tampilkan status agar pengguna tidak mengirim ulang karena mengira sistem macet.
- Jika request gagal, jangan mengklaim transaksi tersimpan.
- Undo mengacu pada transaction ID yang tepat.

# 8. Historical Transaction Intelligence

## 8.1 Tujuan

Pengguna dapat bertanya kapan transaksi terjadi, berapa nominalnya, dan mencari transaksi berdasarkan deskripsi atau kriteria alami.

## 8.2 Contoh Pertanyaan

- "Kapan terakhir kali saya beli sepatu?"
- "Berapa pengeluaran saya untuk kopi minggu lalu?"
- "Kapan saya terakhir bayar internet?"
- "Pernahkah saya membeli laptop bulan lalu?"
- "Tampilkan transaksi tanggal 15 Oktober."
- "Berapa pengeluaran bensin selama tiga bulan terakhir?"
- "Cari transaksi dari BCA minggu ini."
- "Berapa pengeluaran saya di Indomaret?"

## 8.3 Kriteria Pencarian

- Tanggal/rentang tanggal.
- Nominal minimum/maksimum.
- Jenis transaksi.
- Kategori/subkategori.
- Account.
- Merchant.
- Deskripsi/notes.
- Periode relatif: minggu ini, bulan lalu, tahun ini.

## 8.4 Akurasi

- Tanggal dan nominal harus berasal dari query backend.
- Periode relatif dihitung memakai timezone user yang tersimpan.
- Jika tidak ada hasil, AI menyatakan tidak menemukan transaksi cocok.
- Jika banyak hasil mungkin cocok, tampilkan kandidat atau minta klarifikasi.
- Jangan mengarang nominal, tanggal, account, kategori, atau merchant.

# 9. Financial Analyst — Largest Transaction dan Unusual Spending

## 9.1 Largest Transaction

**LOCKED:** dapat diminta melalui Voice atau Chat.

Contoh:
- "Apa pengeluaran terbesar saya bulan ini?"
- "Tampilkan lima transaksi terbesar saya."
- "Kapan terakhir saya mengeluarkan uang lebih dari satu juta?"
- "Apa pembelian paling mahal tahun ini?"
- "Tampilkan transaksi terbesar kategori elektronik."

Fitur:
- Sort berdasarkan amount.
- Filter periode, kategori, account, dan jenis.
- Top N transaction.
- Tampilkan nominal, tanggal, deskripsi, kategori, dan account bila tersedia.

## 9.2 Unusual Spending

**LOCKED:** analisis dilakukan ketika pengguna bertanya; bukan alert proaktif.

Contoh:
- "Ada pengeluaran saya yang tidak biasa bulan ini?"
- "Apakah pengeluaran makanan meningkat?"
- "Ada transaksi yang jauh lebih besar dari kebiasaan saya?"
- "Apakah bulan ini saya lebih boros dibanding bulan lalu?"

Sinyal analisis yang dapat digunakan:
- Historical spending.
- Perbandingan kategori yang sama.
- Median/average transaksi.
- Perubahan frekuensi.
- Perbandingan periode yang sebanding.

Baseline statistik, sample minimum, dan threshold masih OPEN. Bila histori tidak cukup, sistem harus menyatakan keterbatasan. Jangan menyebut transaksi abnormal hanya berdasarkan data yang terlalu sedikit.

## 9.3 Batasan

Proactive Spending Alert tidak termasuk MVP. Sistem tidak mengirim notifikasi otomatis untuk transaksi besar atau tidak biasa.

---

# BAGIAN III — FINANCIAL PLANNING & BUDGET

# 10. Financial Advisor

## 10.1 Prinsip

Financial Advisor membantu pengguna mengelola keuangan sehari-hari. AI memberikan analisis dan pilihan; pengguna tetap mengendalikan perubahan.

## 10.2 Modul

| Modul | Kemampuan | Prioritas |
|---|---|---|
| Budget Planner | Menyusun dan mengevaluasi overall/category budget | Must |
| Spending Optimizer | Menemukan peluang penghematan berdasarkan histori | Must |
| Savings Planner | Target tabungan, kontribusi, dan progres | Should / MVP lanjutan |
| Purchase Advisor | Simulasi kemampuan membeli barang | Should / MVP lanjutan |
| Cash Flow Planning | Perencanaan arus kas ke depan | Future |

## 10.3 Budget Planning

AI dapat:
- Membaca histori income dan expense.
- Mempertimbangkan kewajiban yang sudah diketahui.
- Mengusulkan overall dan category budgets.
- Menyertakan alasan dan data pendukung.
- Membuat draft budget.
- Mengubah draft melalui voice/chat.
- Menampilkan dampak total alokasi.
- Meminta approval sebelum perubahan aktif.

AI tidak boleh mengarang penghasilan, cicilan, kewajiban, atau kebutuhan tetap yang belum dicatat/dikonfirmasi.

## 10.4 Savings Planning

Kemampuan lanjutan:
- Membuat savings goal.
- Menentukan nominal target dan deadline.
- Menghitung kontribusi per periode.
- Menampilkan progress.
- Simulasi perubahan kontribusi/deadline.

Savings goal bukan expense biasa. Model saldo tabungan harus dirancang bersama account model agar tidak terjadi penghitungan ganda.

## 10.5 Spending Optimization

AI dapat menyarankan evaluasi kategori atau kebiasaan pengeluaran, menyertakan periode dan fakta pendukung, serta memperkirakan potensi penghematan. AI tidak boleh mengubah transaksi atau budget tanpa persetujuan.

## 10.6 Purchase Affordability

AI dapat membantu menilai suatu rencana pembelian dengan mempertimbangkan:
- Saldo yang tersedia.
- Pendapatan yang dapat diandalkan.
- Kewajiban pembayaran.
- Pengeluaran rutin.
- Budget yang tersisa.
- Target tabungan yang telah direncanakan.

Saldo total tidak otomatis sama dengan dana bebas pakai. Jika kewajiban atau pemasukan belum lengkap, tampilkan asumsi dan keterbatasan. Hasil berupa simulasi informasional, bukan jaminan kemampuan finansial.

## 10.7 Batas Peran

AI Personal Finance ditujukan untuk financial tracking, budgeting, dan perencanaan keuangan sehari-hari. Rekomendasi bukan nasihat investasi profesional dan tidak menjamin hasil. Investment planning berada di luar MVP.

# 11. Budget Management — Combined Budget

## 11.1 Keputusan yang Dikunci

| Aspek | Keputusan |
|---|---|
| Struktur | Combined Budget: Overall Budget + Category Budget |
| Pembuatan | Hybrid Budget: manual + AI suggested |
| AI management | Conversational Management lewat Voice/Chat |
| Perubahan oleh AI | Proposal/draft terlebih dahulu |
| Persetujuan | User approval wajib sebelum diterapkan |
| Budget terlampaui | Warning Only; transaksi tetap boleh dicatat |

## 11.2 Konsep

- **Overall Budget:** batas total expense dalam satu periode.
- **Category Budget:** batas expense untuk kategori tertentu.
- **Unallocated Budget:** bagian Overall Budget yang belum dialokasikan ke kategori.
- Expense kategori tanpa alokasi tetap dihitung pada Overall Budget.

## 11.3 Aturan Alokasi

Aturan yang diusulkan:

`Unallocated = Overall Budget - Sum(Category Budgets)`

Category allocation tidak boleh melebihi Overall Budget. Ini merupakan validation rule yang direkomendasikan dan perlu dikonfirmasi saat technical design.

## 11.4 Perhitungan

`Overall Remaining = Overall Budget - Actual Expense`

`Category Remaining = Category Limit - Actual Category Expense`

- Income tidak mengurangi budget.
- Transfer internal tidak dihitung sebagai expense.
- Draft dan transaksi voided tidak dihitung.
- Expense valid dihitung pada Overall Budget.
- Expense berkategori dihitung pada Category Budget terkait.
- Uncategorized expense tetap masuk Overall Budget.
- Refund perlu dikaitkan dengan transaksi asal jika memungkinkan; definisi gross/net expense harus konsisten.

## 11.5 Warning Only

**LOCKED:** jika budget terlampaui:
- Transaksi tetap disimpan.
- Sistem tidak menolak transaksi.
- Tidak ada approval tambahan hanya karena budget exceeded.
- Budget tidak berubah otomatis.
- Sistem menampilkan status dan nominal kelebihan.
- AI dapat menjelaskan penyebab atau memberi evaluasi bila diminta.

Threshold awal yang diusulkan:

| Pemakaian | Status |
|---|---|
| Di bawah 80% | Normal |
| 80% sampai kurang dari 100% | Warning |
| 100% atau lebih | Exceeded |

Ambang 80% masih PROPOSED, bukan keputusan final.

## 11.6 Period dan Rollover

- Default bulanan direkomendasikan untuk MVP.
- Tidak ada automatic rollover yang direkomendasikan.
- Periode khusus atau tanggal mulai kustom dapat dipertimbangkan.
- Kedua aturan ini belum LOCKED.

## 11.7 Conversational Budget Management

Contoh:
- "Buat budget pengeluaran 3 juta untuk bulan depan."
- "Buat budget makanan 1 juta."
- "Ubah budget transportasi menjadi 600 ribu."
- "Kurangi budget hiburan 100 ribu dan alihkan ke makanan."
- "Bagaimana kondisi budget saya saat ini?"
- "Kenapa budget tagihan saya hampir habis?"

Alur:
1. Intent dan parameter dikenali.
2. Backend mengambil budget aktif dan data relevan.
3. AI membuat proposal/draft.
4. Validation service memeriksa alokasi dan aturan bisnis.
5. Pengguna melihat ringkasan before/after dan dampak.
6. Pengguna approve, edit, atau reject.
7. Backend memeriksa ulang user ownership dan budget version.
8. Perubahan diterapkan secara atomik dalam database transaction.
9. Audit log dibuat.
10. Assistant mengonfirmasi hasil hanya setelah backend sukses.

Approval harus spesifik untuk proposal. Approval satu proposal tidak dapat digunakan untuk mutasi lain. Proposal kedaluwarsa atau berbasis budget version lama harus divalidasi ulang.

## 11.8 Budget UI

- Period selector.
- Overall Budget Card.
- Actual spending dan remaining.
- Category allocation list.
- Progress/status.
- Unallocated budget.
- AI Budget Insight.
- Manage Budget / Ask AI.
- Pending proposals.
- Budget history/audit trail.

# 12. Dashboard, Reports, dan Analytics

## 12.1 Dashboard

Komponen:
- Total balance (bergantung account model).
- Total income periode berjalan.
- Total expense periode berjalan.
- Net cash flow.
- Cash flow chart.
- Expense breakdown by category.
- Recent transactions.
- Budget snapshot.
- Akses cepat ke Assistant.

## 12.2 Reports

MVP:
- Income vs Expense.
- Net Cash Flow.
- Spending by Category.
- Monthly Spending Trend.
- Periode filter.

Recharts menjadi rekomendasi library chart untuk frontend React; finalisasi saat technical spike.

## 12.3 Data Rules

- Agregasi angka dilakukan backend/database.
- Transfer dikecualikan dari income dan expense.
- Draft/voided dikecualikan.
- Kebijakan refund dan gross/net expense harus konsisten.
- Periode mengikuti timezone pengguna.
- Ketika transaksi tersimpan atau diubah, query/cache dashboard dan reports harus diperbarui.

---

# BAGIAN IV — TECHNICAL REQUIREMENTS

# 13. Core Technology Stack

## 13.1 Stack yang Dipilih

| Layer | Teknologi | Status | Tanggung jawab |
|---|---|---|---|
| Frontend framework | **Next.js** (App Router) | LOCKED | Responsive web application |
| Frontend library | React + TypeScript | LOCKED sebagai fondasi Next.js | UI dan interaksi |
| Backend language | **Go / Golang** | LOCKED | API, domain logic, financial calculations, AI orchestration |
| Backend HTTP framework | **Gin** | LOCKED | Routing, middleware, HTTP request/response |
| Main database | **PostgreSQL** | LOCKED | Source of truth data keuangan |
| Container tools | **Docker + Docker Compose** | LOCKED | Local environment dan container packaging |

Core stack: **Next.js + Go/Gin + PostgreSQL + Docker**.

Tidak menggunakan NestJS sebagai backend kedua. Tidak perlu menjalankan Gin, Chi, dan Echo secara bersamaan.

## 13.2 Supporting Technologies (PROPOSED)

| Area | Rekomendasi | Catatan |
|---|---|---|
| Styling | Tailwind CSS + shadcn/ui | Komponen dapat dikustomisasi |
| Server state | TanStack Query | Fetching, caching, invalidation |
| Form | React Hook Form + Zod | Validasi frontend; backend tetap validasi ulang |
| Charts | Recharts | Dashboard dan analytics |
| Go PostgreSQL driver | pgx | Koneksi PostgreSQL |
| Typed SQL | sqlc | Generate typed Go code dari query SQL |
| Migration | goose, golang-migrate, Atlas, atau setara | Pilih satu melalui TDD |
| AI provider | OpenAI API | Kandidat model/tool calling |
| AI Go SDK | Official OpenAI Go SDK | Akses API dari backend Go |
| AI chat UI | Vercel AI SDK UI atau custom | Opsional pada frontend; backend tetap Go |
| Speech capture | Browser MediaRecorder API | Audio capture dari microphone |
| STT | OpenAI Transcription API atau provider setara | Harus diuji Bahasa Indonesia |
| Realtime voice | OpenAI Realtime API atau provider setara | Future, setelah voice dasar stabil |
| Authentication | Supabase Auth atau managed provider setara | Belum dipilih final |
| Testing | Go testing + integration tests + Playwright | Unit, API, browser, E2E |
| Cache/queue | Redis + Go-compatible worker/queue library | Future; tidak perlu pada fondasi MVP awal |

Prisma tidak digunakan sebagai ORM backend utama karena backend memakai Go. `pgx + sqlc` menjadi opsi rekomendasi, tetapi perlu technical spike dan keputusan final di TDD.

## 13.3 Version Policy

- Gunakan release stable yang kompatibel pada saat repository dibuat.
- Pin versi toolchain dan dependency di file project.
- Jangan menjadikan nomor patch pada PRD sebagai komitmen jangka panjang.
- Periksa compatibility matrix Next.js/React, Go/Gin, PostgreSQL driver, dan SDK sebelum upgrade.
- Hindari Release Candidate, Beta, Canary, atau Alpha untuk dependency production kecuali ada justifikasi dan persetujuan.

# 14. Docker dan Local Development

## 14.1 Kondisi

**LOCKED:** Docker sudah tersedia pada komputer pengguna. Tidak diperlukan instruksi instalasi Docker sebagai bagian dari project setup; gunakan instalasi yang sudah ada.

## 14.2 Tujuan Penggunaan

Docker digunakan untuk:
- Menjalankan service development dalam environment konsisten.
- Mengisolasi dependensi.
- Menjalankan PostgreSQL lokal.
- Mengelola jaringan antar-container.
- Menyediakan persistent volume database.
- Membuat image frontend/backend untuk deployment.
- Menyederhanakan setup developer lain.

Docker bukan framework aplikasi, bukan pengganti Gin/Next.js, dan tidak berarti seluruh layanan eksternal perlu dijalankan di container.

## 14.3 Service Awal

**MVP local development:**
- PostgreSQL dalam container dengan persistent volume.
- Next.js dan Go API boleh dijalankan langsung di host untuk hot reload, atau di container bila konfigurasi sudah siap.
- Docker Compose mengatur service pendukung.

**Future:**
- Redis jika cache/queue diperlukan.
- Worker container jika background job telah diimplementasikan.

OpenAI API dan Supabase Cloud (jika dipilih) adalah layanan eksternal, bukan container lokal.

## 14.4 File/Artefak Docker

- `docker-compose.yml` untuk local multi-service orchestration.
- `apps/web/Dockerfile` untuk Next.js saat containerized deployment diperlukan.
- `apps/api/Dockerfile` untuk Go API.
- `.dockerignore`.
- `.env.example` tanpa secret.
- Persistent volume untuk PostgreSQL local development.
- Health check untuk service yang bergantung pada database.

## 14.5 Operasional

Perintah dasar setelah konfigurasi project tersedia:

```bash
docker compose up -d
docker compose ps
docker compose logs -f
docker compose down
```

`docker compose down` tidak boleh disamakan dengan penghapusan volume. Data database lokal harus tetap persisten kecuali pengguna secara eksplisit melakukan reset volume.

---

# BAGIAN V — ARCHITECTURE, API, SECURITY

# 15. High-level Architecture

```text
Mobile / Tablet / Desktop Browser
              |
              v
         Next.js Web
              |
       HTTPS / API / Stream
              |
              v
        Go + Gin Backend
              |
  +-----------+------------+
  |           |            |
  v           v            v
Financial   AI Tool      Speech
Modules     Orchestrator Service
  |           |            |
  +-----------+------------+
              |
              v
          PostgreSQL
```

## 15.1 Architecture Style

**PROPOSED:** Modular Monolith untuk backend awal.

- Satu deployable Go API.
- Modul domain terpisah.
- Business logic tidak berada di Gin handler.
- Repository/database access terpisah dari service.
- AI tool memanggil domain service yang sama dengan jalur manual.
- Service tertentu dapat dipisah di masa depan berdasarkan kebutuhan operasional yang nyata.

Microservices tidak diperlukan untuk MVP.

## 15.2 Backend Module Candidate

```text
apps/api/
├── cmd/api/main.go
├── internal/
│   ├── modules/
│   │   ├── auth/
│   │   ├── users/
│   │   ├── accounts/
│   │   ├── categories/
│   │   ├── transactions/
│   │   ├── budgets/
│   │   ├── savings/
│   │   ├── analytics/
│   │   ├── assistant/
│   │   └── financial_advisor/
│   ├── middleware/
│   ├── infrastructure/
│   │   ├── database/
│   │   ├── ai/
│   │   └── storage/
│   └── shared/
├── db/
│   ├── migrations/
│   ├── queries/
│   └── generated/
├── go.mod
└── go.sum
```

Pembagian file setiap modul dapat menggunakan handler, service, repository, model, dan DTO sesuai kebutuhan; jangan membuat abstraksi berlebihan untuk modul sederhana.

## 15.3 Frontend Project Structure Candidate

```text
apps/web/
├── app/
├── components/
│   ├── navigation/
│   ├── assistant/
│   ├── transaction/
│   ├── dashboard/
│   └── shared/
├── features/
│   ├── auth/
│   ├── accounts/
│   ├── transactions/
│   ├── budgets/
│   ├── analytics/
│   └── assistant/
├── hooks/
├── lib/
└── public/
```

Struktur ini adalah starting point. Organisasi akhir dapat berkembang berdasarkan feature boundary.

# 16. AI Tool Gateway

## 16.1 Prinsip

- LLM tidak diberi akses SQL langsung.
- Semua operasi membaca data melalui query/service backend yang terkontrol.
- Semua mutasi melalui business service tervalidasi.
- `user_id` berasal dari authentication context backend, bukan parameter yang bebas ditentukan model.
- Tool input memakai schema yang ketat.
- Setiap mutasi memeriksa authorization dan business rules.
- Operasi yang dapat terduplikasi menggunakan idempotency mechanism.
- Operasi yang butuh persetujuan mengacu pada proposal ID tertentu.
- Mutasi penting menghasilkan audit log.

## 16.2 Candidate Tools

| Tool | Fungsi |
|---|---|
| `create_transaction` | Membuat transaksi valid |
| `search_transactions` | Mencari riwayat transaksi |
| `update_transaction` | Mengubah transaksi setelah target tervalidasi |
| `void_transaction` | Membatalkan transaksi dengan audit trail |
| `get_financial_summary` | Mengambil ringkasan keuangan |
| `get_largest_transactions` | Mencari transaksi terbesar |
| `detect_unusual_spending` | Analisis unusual spending saat diminta |
| `get_budget_status` | Mengambil status overall/category budget |
| `create_budget_proposal` | Membuat draft perubahan budget |
| `update_budget_proposal` | Mengubah draft proposal |
| `approve_budget_proposal` | Menerapkan proposal spesifik yang disetujui |
| `reject_budget_proposal` | Menolak proposal |
| `get_savings_analysis` | Analisis target tabungan bila modul tersedia |

Nama tool merupakan kontrak logis; nama teknis dapat disesuaikan tanpa mengubah perilaku produk.

# 17. Candidate API Specification

Endpoint berikut merupakan starting point untuk OpenAPI/TDD.

```http
# Authentication / profile
GET    /api/v1/me

# Accounts
GET    /api/v1/accounts
POST   /api/v1/accounts
PATCH  /api/v1/accounts/{id}

# Categories
GET    /api/v1/categories
POST   /api/v1/categories
PATCH  /api/v1/categories/{id}

# Transactions
POST   /api/v1/transactions
GET    /api/v1/transactions
GET    /api/v1/transactions/{id}
PATCH  /api/v1/transactions/{id}
POST   /api/v1/transactions/{id}/void
GET    /api/v1/transactions/search

# Analytics
GET    /api/v1/analytics/financial-summary
GET    /api/v1/analytics/largest-transactions
GET    /api/v1/analytics/unusual-spending
GET    /api/v1/analytics/spending-comparison
GET    /api/v1/analytics/cash-flow

# Budgets
GET    /api/v1/budgets
POST   /api/v1/budgets
GET    /api/v1/budgets/{id}
PATCH  /api/v1/budgets/{id}
GET    /api/v1/budgets/{id}/status

# Budget proposals
POST   /api/v1/budget-proposals
GET    /api/v1/budget-proposals/{id}
PATCH  /api/v1/budget-proposals/{id}
POST   /api/v1/budget-proposals/{id}/approve
POST   /api/v1/budget-proposals/{id}/reject

# Assistant and speech
POST   /api/v1/assistant/messages
POST   /api/v1/speech/transcriptions
```

Final API method, pagination, error schema, streaming protocol, idempotency header, and naming conventions harus ditetapkan dalam API specification sebelum implementasi.

---

# BAGIAN VI — UI/UX DESIGN REQUIREMENTS

# 18. Visual Design Direction

## 18.1 Status

**LOCKED:** mobile-first, responsive, clean and usable interaction, voice primary action.  
**OPEN:** brand, visual style final, palette, light/dark mode, typography.

Arah visual yang disarankan:
- Modern minimalist / modern fintech.
- Tata letak bersih dan tidak padat.
- Whitespace yang cukup.
- Hierarki angka keuangan jelas.
- Penggunaan warna bermakna dan konsisten.
- Card untuk informasi ringkas; tabel lebih cocok untuk desktop.

Arah warna sementara (belum final):
- Green untuk positive/success/income emphasis.
- Red untuk expense/negative.
- Blue untuk transfer/informational.
- Amber untuk warning.
- Neutral background dan surface untuk layout.

## 18.2 Responsive Layout

Breakpoints kandidat (perlu disesuaikan saat implementasi):

| Viewport | Layout |
|---|---|
| Mobile < 768px | Single column, bottom navigation, assistant full-screen |
| Tablet 768–1023px | Adaptive columns, navigation sesuai ruang |
| Desktop >= 1024px | Sidebar dan multi-column dashboard |

Tidak ada horizontal overflow pada viewport yang didukung.

## 18.3 Touch dan Accessibility

- Target sentuh minimal sekitar 44 × 44 CSS px.
- Kontras teks dan kontrol memadai.
- Informasi tidak disampaikan hanya melalui warna.
- Mendukung keyboard navigation desktop.
- Memperhatikan safe area smartphone.
- Input tidak tertutup virtual keyboard.
- Loading, empty, error, permission, success, dan cancelled state dirancang.
- Grafik memiliki label/summary yang terbaca tanpa hover.

# 19. Detail UI per Screen

## 19.1 Home Dashboard

Urutan konten mobile yang disarankan:
1. Greeting, tanggal, profile, notification.
2. Total balance card (setelah account model final).
3. Income dan expense periode berjalan.
4. Cash flow chart.
5. Expense category breakdown.
6. Recent transactions.
7. Budget snapshot dan AI insight.

Desktop dapat mengubah konten menjadi multi-column. Angka penting harus mudah ditemukan tanpa scrolling berlebihan.

## 19.2 History

Komponen:
- Search bar.
- Date/period filter.
- Type filter: income, expense, transfer.
- Category filter.
- Account filter.
- Grouping by date.
- Transaction list/card di mobile.
- Data table di desktop.
- Transaction detail bottom sheet di mobile.
- Edit dan void/Undo action.
- Source label (manual, voice, chat) bila berguna.

## 19.3 Assistant Full-screen

Komponen:
- Header: Back, assistant name/status, conversation history.
- Conversation area.
- Suggested prompts.
- Voice recorder.
- Audio waveform saat listening.
- Chat input.
- Transaction result card.
- Financial analysis card.
- Clarification card.
- Budget proposal/approval card.

Chat input harus tetap tersedia sebagai alternatif voice.

## 19.4 Assistant States

| State | UI behavior |
|---|---|
| Ready | Microphone siap; instruksi jelas |
| Permission Required | Jelaskan izin; Chat tetap dapat digunakan |
| Listening | Indikator audio nyata dan transkripsi sementara bila didukung |
| Processing | Status pemrosesan; cegah double submit |
| Clarification | Pertanyaan dan pilihan/voice/chat response |
| Success | Hasil transaksi/analisis |
| Error | Penjelasan dan retry/fallback |
| Cancelled | Recording dibatalkan tanpa menyimpan |

Jangan menampilkan status Listening sebelum browser menerima audio. Jangan mengulang recording otomatis setelah error.

## 19.5 Conversation Cards

Jenis card:
- **Transaction Result:** amount, type, category, account, date, source, Undo.
- **Financial Summary:** income, expense, net cash flow.
- **Historical Search:** tanggal, nominal, description, account, kategori.
- **Largest Transaction:** Top N list.
- **Unusual Spending:** temuan, pembanding, alasan, keterbatasan sampel.
- **Budget Status:** actual, limit, remaining, warning.
- **Budget Proposal:** before/after, impact, approve/edit/reject.
- **Savings Plan:** target, contribution, progress (bila modul tersedia).

## 19.6 Reports

- Summary cards.
- Income vs expense chart.
- Cash flow trend.
- Category breakdown.
- Period selector.
- Detail transaksi terkait bila pengguna ingin menelusuri sumber angka.

Chart tidak boleh terlalu padat pada mobile. Gunakan tooltip/selection yang ramah touch.

## 19.7 Budget Screen

- Period selector.
- Overall budget card.
- Actual spending, remaining, percentage.
- Category allocation and usage.
- Unallocated budget.
- AI budget insight.
- Button/link untuk Manage Budget with AI.
- Pending proposals.
- Budget audit/history.

## 19.8 Profile / Settings

- Personal information.
- Accounts/wallets.
- Categories.
- Currency/locale/timezone.
- AI preferences.
- Microphone permissions help.
- Notification preferences.
- Privacy and security.

# 20. Reusable UI Components

Candidate components:

```text
components/
├── navigation/
│   ├── BottomNavigation
│   ├── AssistantActionButton
│   └── DesktopSidebar
│
├── assistant/
│   ├── AssistantHeader
│   ├── ConversationList
│   ├── MessageBubble
│   ├── VoiceRecorder
│   ├── AudioWaveform
│   ├── ChatInput
│   ├── TransactionResultCard
│   ├── FinancialAnalysisCard
│   ├── ClarificationCard
│   └── BudgetProposalCard
│
├── transaction/
│   ├── TransactionItem
│   ├── TransactionDetailSheet
│   ├── TransactionFilters
│   └── TransactionForm
│
├── dashboard/
│   ├── BalanceCard
│   ├── CashFlowChart
│   ├── CategoryBreakdown
│   └── RecentTransactions
│
├── budget/
│   ├── OverallBudgetCard
│   ├── CategoryBudgetItem
│   ├── BudgetProgress
│   ├── BudgetWarning
│   └── BudgetProposalReview
│
└── shared/
    ├── EmptyState
    ├── LoadingState
    ├── ErrorState
    ├── ConfirmationDialog
    └── PeriodSelector
```

Gunakan design tokens dan reusable components; jangan membuat komponen yang sama berulang dengan visual/perilaku berbeda tanpa alasan.

---

# BAGIAN VII — NON-FUNCTIONAL, TESTING, DELIVERY

# 21. Non-Functional Requirements

## 21.1 Security & Privacy

- Semua endpoint memerlukan authentication kecuali endpoint publik yang memang diperlukan.
- Authorization memeriksa ownership setiap resource.
- Setiap query dibatasi oleh user identity yang sah.
- PostgreSQL Row-Level Security dapat digunakan sebagai defense-in-depth jika dirancang dan dites dengan benar.
- API secret, database credential, dan service-role key tidak boleh berada di browser.
- Jangan mengirim informasi pribadi lebih banyak dari yang dibutuhkan ke provider AI.
- Definisikan retention audio, transcript, conversation, dan audit log.
- Sediakan mekanisme export/delete data sesuai kebijakan produk dan hukum yang berlaku.
- Terapkan rate limit untuk authentication, speech, AI, dan endpoint mutasi.

## 21.2 Financial Data Integrity

- Semua nominal diproses dengan tipe integer/decimal yang tepat; hindari floating-point.
- Saldo dan agregasi dihitung backend/database.
- Transfer harus seimbang.
- Draft dan voided transaction tidak masuk laporan.
- Mutasi multi-record yang harus konsisten menggunakan database transaction.
- Budget approval melakukan version/ownership re-check.
- Idempotency diterapkan pada mutasi yang rentan retry.
- Audit trail untuk perubahan transaksi dan budget.

## 21.3 Reliability

- Jangan klaim sukses sebelum backend mengonfirmasi.
- Tangani timeout, jaringan terputus, retry, dan duplicate request.
- Sediakan Chat fallback jika voice gagal.
- Tampilkan status processing yang akurat.
- Persistent volume database development tidak boleh terhapus tanpa aksi eksplisit.

## 21.4 Observability

PROPOSED:
- Structured application logs.
- Correlation/request IDs.
- Error tracking.
- Metrics untuk latency API, database, STT, AI provider, dan error rate.
- Audit logs terpisah dari operational logs.

## 21.5 Performance

Target numerik belum ditetapkan. Sistem harus:
- Menggunakan pagination untuk transaction history.
- Mengoptimalkan aggregate queries.
- Menambahkan index pada filter yang sering dipakai setelah query patterns terukur.
- Menghindari mengambil seluruh transaksi untuk menjawab pertanyaan agregat.
- Menggunakan streaming untuk AI bila pengalaman pengguna membutuhkannya.
- Menguji performa dengan dataset realistis.

# 22. Testing Strategy

## 22.1 Backend Go

- Unit tests untuk service dan business rules.
- Integration tests terhadap PostgreSQL.
- Test transfer balance consistency.
- Test budget calculation dan over-budget behavior.
- Test idempotency.
- Test user ownership dan cross-user access denial.
- Test budget proposal approval, rejection, stale version, dan atomicity.

## 22.2 Frontend Next.js

- Component tests untuk navigasi dan Assistant state.
- Form validation tests.
- Responsive layout tests.
- Keyboard/safe-area behavior tests.
- Voice permission dan error state tests.

## 22.3 AI Evaluation

Buat test cases dalam Bahasa Indonesia untuk:
- Penulisan nominal: 25 ribu, 150 ribu, 1,2 juta, 2 juta 500 ribu.
- Tanggal: hari ini, kemarin, tanggal eksplisit, bulan lalu.
- Multi-transaction dalam satu kalimat.
- Code-switching dan bahasa informal.
- Transfer vs expense.
- Pertanyaan vs perintah transaksi.
- Koreksi transaksi sebelumnya.
- Account/category ambiguity.
- Largest Transaction dan Unusual Spending.
- Budget proposal dan approval.

Ukur transcription accuracy, intent accuracy, entity extraction, duplicate rate, clarification correctness, dan numerical answer accuracy.

## 22.4 End-to-End Scenarios

1. Register → create account → manual expense → dashboard updated.
2. Voice expense → STT → parse → validation → transaction saved.
3. Ambiguous voice → draft → clarification → transaction saved.
4. Duplicate retry → hanya satu transaction tersimpan.
5. Ask largest expense → hasil cocok dengan query database.
6. Ask unusual spending dengan data kurang → AI menyatakan keterbatasan.
7. Create budget proposal → approval → budget updated + audit log.
8. Over-budget transaction → tetap tersimpan + warning.
9. User A mencoba membaca data User B → access denied.
10. Mobile Assistant dibuka → bottom navigation tersembunyi → Back mengembalikan ke halaman sebelumnya.

# 23. Acceptance Criteria — MVP Exit Criteria

MVP dapat masuk pilot setelah:

1. User dapat register/login dan data antar-user terisolasi.
2. User dapat membuat/mengelola account sesuai model account yang diputuskan.
3. User dapat mencatat income, expense, dan transfer secara manual.
4. User dapat melihat, mencari, mengedit, dan void transaksi.
5. User dapat mencatat transaksi lewat Voice dan Chat.
6. Hybrid Recording membedakan input valid dan ambigu.
7. Draft tidak mengubah saldo, laporan, atau budget.
8. Retry tidak menghasilkan transaksi ganda.
9. Dashboard dan reports menggunakan angka backend yang konsisten.
10. AI dapat menjawab pertanyaan historis dengan tanggal dan nominal yang cocok dengan database.
11. AI dapat menampilkan Largest Transaction.
12. AI dapat menganalisis Unusual Spending saat diminta dan menyatakan jika data tidak cukup.
13. Sistem mendukung Combined Budget.
14. AI dapat membuat dan mengubah budget proposal melalui percakapan.
15. Budget change memerlukan approval dan audit log.
16. Over-budget tidak memblokir transaksi dan menghasilkan warning.
17. Mobile memiliki lima bottom navigation items dan Assistant full-screen.
18. Voice error/permission denial memiliki Chat fallback.
19. Security test membuktikan tidak ada akses lintas-user.
20. Core flows lulus integration dan end-to-end tests.

---

# BAGIAN VIII — ROADMAP DAN OPEN DECISIONS

# 24. Implementation Roadmap

Frontend dan backend dapat dikembangkan paralel melalui API contract/mock data. AI tidak perlu diselesaikan terlebih dahulu. Core financial system harus tersedia agar AI mempunyai data dan business services yang benar.

| Phase | Fokus | Output |
|---|---|---|
| 0 | Product/Technical Design | TDD, ERD, API contract, design tokens, decision log |
| 1 | Foundation | Monorepo, Next.js, Go/Gin, PostgreSQL, Docker Compose, auth, migrations |
| 2 | Financial Core | Accounts, categories, transactions, transfer, history |
| 3 | Dashboard & Budget Core | Dashboard, reports, Combined Budget, warning behavior, approval domain |
| 4 | AI Transaction Capture | Chat, audio recording, STT, intent parsing, Hybrid Recording |
| 5 | Financial Intelligence | Historical search, Largest Transaction, Unusual Spending, Financial Analyst |
| 6 | Financial Advisor | Budget suggestions, conversational budget, savings/purchase planning sesuai prioritas |
| 7 | Pilot & Production | Security, AI evaluation, performance, monitoring, deployment |

Urutan dapat berubah berdasarkan dependency, tetapi AI tidak boleh menduplikasi atau melewati financial business logic.

# 25. Open Decisions yang Harus Ditutup Sebelum Fitur Terkait

| ID | Keputusan | Mengapa penting | Rekomendasi awal |
|---|---|---|---|
| O-01 | Account model | Saldo, transfer, dan purchase affordability | Hybrid Accounts |
| O-02 | Brand/name | Branding dan product identity | Tentukan setelah visual exploration |
| O-03 | Visual system | Warna, theme, typography | Modern minimalist; user confirmation |
| O-04 | Authentication provider | Session/JWT lifecycle | Evaluasi Supabase Auth vs custom/managed |
| O-05 | Database access | Query, migrations, generated code | pgx + sqlc technical spike |
| O-06 | AI provider/model | Akurasi, latency, biaya, privacy | OpenAI API sebagai kandidat utama |
| O-07 | STT provider | Voice accuracy Bahasa Indonesia | Benchmark beberapa perintah nyata |
| O-08 | AI transcript retention | Privasi dan debugging | Data minimization; tentukan retention |
| O-09 | Unusual-spending baseline | Menghindari false positive | Uji median dan minimum sample |
| O-10 | Budget period | Perhitungan period | Bulanan default |
| O-11 | Budget rollover | Sisa limit periode | Tidak otomatis rollover |
| O-12 | Budget warning threshold | Kapan warning muncul | 80% sebagai usulan |
| O-13 | Refund accounting | Gross/net expense | Link refund ke original transaction |
| O-14 | Debt domain | Pinjaman dan cicilan | Di luar MVP hingga rules didefinisikan |
| O-15 | Currency support | Presisi dan format uang | IDR awal |
| O-16 | Conversation retention | Riwayat Assistant | Tentukan periode, export, delete |
| O-17 | Savings account model | Hindari double counting | Kaitkan goal dengan account/contribution model |

# 26. Product Guardrails

1. AI tidak boleh mengarang financial data.
2. AI tidak boleh menjalankan SQL secara langsung.
3. AI tidak boleh mengubah budget tanpa approval.
4. AI tidak boleh mengubah account balance secara langsung; saldo diturunkan dari ledger/transactions sesuai aturan domain.
5. Budget exceeded tidak memblokir transaksi.
6. Transfer internal tidak dihitung sebagai income/expense.
7. Draft tidak dihitung sebagai transaksi valid.
8. Unusual Spending bukan proactive alert pada MVP.
9. Approval harus spesifik terhadap proposal dan data version tertentu.
10. Pengguna tetap dapat melakukan pencatatan manual jika AI/voice tidak tersedia.

# 27. Glossary

| Istilah | Arti |
|---|---|
| Hybrid Recording | Transaksi jelas dapat disimpan otomatis; transaksi ambigu meminta klarifikasi |
| Smart Assistant | Satu pengalaman AI untuk voice dan chat |
| Full-screen Assistant | Halaman Assistant mobile tanpa Bottom Navigation |
| Historical Transaction Intelligence | Pencarian transaksi masa lalu menggunakan bahasa alami |
| Largest Transaction | Pencarian transaksi bernominal terbesar |
| Unusual Spending | Analisis pengeluaran yang tidak biasa berdasarkan histori saat diminta |
| Overall Budget | Batas total pengeluaran dalam periode |
| Category Budget | Batas pengeluaran untuk kategori |
| Combined Budget | Overall Budget dan Category Budget digunakan bersama |
| Unallocated Budget | Bagian Overall Budget yang belum dialokasikan ke kategori |
| Warning Only | Transaksi tetap boleh dicatat meskipun melebihi budget |
| Budget Proposal | Draft perubahan budget yang menunggu keputusan pengguna |
| Financial Context Engine | Backend service yang menyediakan data finansial relevan dan tervalidasi kepada AI |
| Idempotency | Mekanisme agar request ulang tidak menggandakan mutasi |

---

# 28. Pernyataan Baseline

Dokumen ini menjadi **single source of truth PRD AI Personal Finance v1.2** untuk memulai tahap Technical Design Document, database schema/ERD, API specification, UI design, dan implementation backlog.

Perubahan terhadap requirement `LOCKED` harus dicatat sebagai product decision/change log. Item `PROPOSED` dan `OPEN` tidak boleh dipaksakan menjadi keputusan final tanpa persetujuan product owner.

**Core stack yang digunakan:** Next.js + Go/Gin + PostgreSQL + Docker (Docker sudah tersedia dan akan langsung digunakan).  
**Strategi implementasi:** Financial Core terlebih dahulu, Frontend dan Backend berjalan paralel, lalu AI diintegrasikan melalui backend business services yang sama.

**End of Document — AI Personal Finance PRD v1.2**
