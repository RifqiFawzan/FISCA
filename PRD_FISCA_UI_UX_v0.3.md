# UI/UX Design PRD — FISCA (AI Personal Finance)

**Versi:** 0.3 (Updated with Final Visual Identity & Eye Ergonomics Design System)  
**Tanggal:** 8 November 2026  
**Status:** Approved Visual Direction & Locked Product Specs  
**Jenis Dokumen:** Product Requirements Document khusus UI/UX  
**Nama Produk:** FISCA  
**Design System Baseline:** Luminescent Carbon Fintech (Dark-First & Velvety Glassmorphism)

---

## 1. Ringkasan Eksekutif Produk

**FISCA** adalah aplikasi web pencatat dan pengelola keuangan pribadi cerdas berbasis AI yang dirancang dengan pendekatan **mobile-first responsive web**. Pengguna dapat mencatat arus kas (pemasukan, pengeluaran, transfer) melalui interaksi suara (voice command) atau teks (chat), memantau ringkasan finansial harian, mengelola anggaran terpadu (*Combined Budget*), menganalisis laporan berkala, serta berinteraksi langsung dengan AI Financial Assistant.

Aplikasi mengusung bahasa visual resmi **Dark-First Carbon-Aware & Frosted Velvet Glassmorphism** yang dioptimalkan untuk **kenyamanan mata (Eye Ergonomics & Visual Fatigue Optimization)**, menjadikannya ramah di layar OLED dan nyaman digunakan dalam kondisi pencahayaan rendah (malam hari) tanpa efek silau (*zero optical glare*).

---

## 2. Keputusan Desain & Brand Identity yang Dikunci (Locked Decisions)

| Area | Keputusan Final | Keterangan & Rincian |
|---|---|---|
| **Nama Aplikasi** | **FISCA** | Financial Intelligence, Smart Carbon & Allocation |
| **Pendekatan Desain** | **Mobile-First Responsive Web** | Fokus utama layar smartphone (~390px viewport), adaptif ke desktop/tablet |
| **Arah Visual (Theme)** | **Dark-First Frosted Velvet Glassmorphism** | Permukaan obsidian gelap, blur kaca halus (*backdrop-blur*), specular hairline border |
| **Aksen & Palet Warna** | **Luminescent Carbon Emerald & Slate** | Emerald luminescent lembut (`#10b981`, `#059669`, `#047857`) dipadu background obsidian dalam (`#0a0e16`, `#0f131c`) |
| **Standar Ergonomi Visual** | **Eye Ergonomics (Anti-Glare / Anti-Fatigue)** | Teks off-white (`#f1f5f9`), teks sekunder slate (`#94a3b8`), ambient diffused glow tanpa harsh neon glare |
| **Tipografi** | **Plus Jakarta Sans** | Modern geometric sans-serif dengan keterbacaan angka (*tabular figures*) yang presisi |
| **Navigasi Mobile** | **5-Item Bottom Navigation** | Home, History, **FISCA Assistant (Elevated Center)**, Reports, Profile |
| **Mode Assistant** | **Full-Screen AI Experience** | Bottom Navigation disembunyikan; Voice waveform + conversational rich cards |
| **Pencatatan Transaksi** | **Hybrid Recording UX** | Voice & Chat multimodal: auto-save untuk input lengkap, clarification card untuk input ambigu |
| **Model Budget** | **Combined Budget** | Overall Budget + Category Budgets; Status: Normal (<80%), Warning (80-100%), Exceeded (>100%) |
| **Kebijakan Over-Limit** | **Warning Only** | Transaksi tetap dicatat; tidak memblokir pencatatan riil pengguna |
| **Perubahan Budget via AI** | **Proposal & Approval System** | AI tidak mengubah nominal aktif secara sepihak; harus melalui kartu persetujuan pengguna |
| **Autentikasi & Keamanan** | **Biometric & Bank-Grade Security** | Single Sign-On (Google/Apple), Face ID & Fingerprint ready, enkripsi lokal AES-256 / SHA-256 |

---

## 3. Spesifikasi Visual & Design System (Luminescent Carbon Fintech)

### 3.1 Palet Warna & Kontras Ramah Mata (Eye Ergonomics)

1. **Latar Belakang & Permukaan Kaca (Surfaces):**
   - `surface-canvas-lowest`: `#0a0e16` (Deep obsidian black, hemat daya layar OLED).
   - `surface-base`: `#0f131c` (Dark slate-tinted obsidian).
   - `surface-container-low`: `#181c24` (Subtle charcoal container).
   - `surface-glass`: `rgba(24, 28, 36, 0.70)` dengan `backdrop-filter: blur(20px)` dan border specular `rgba(255, 255, 255, 0.06)`.

2. **Tipografi & Kontras Retina (Anti-Fatigue Contrast):**
   - **Teks Utama (Headings & Nominal):** Slate Off-White (`#f1f5f9` / `#e2e8f0`) — mengeliminasi ketegangan mata akibat kontras putih 100% mentah (`#ffffff`).
   - **Teks Sekunder & Caption:** Muted Zinc / Cool Slate (`#94a3b8` / `#64748b`) — rasio kontras memenuhi WCAG AAA tanpa distracting.
   - **Border & Pembagi (Dividers):** Specular hairline borders `rgba(255, 255, 255, 0.05)` hingga `rgba(255, 255, 255, 0.08)`.

3. **Aksen Interaktif (Brand Luminescence):**
   - **Primary Action (CTA Buttons):** Velvety Emerald Luminescent Gradient (`linear-gradient(135deg, #047857 0%, #059669 50%, #10b981 100%)`).
   - **Drop Shadow & Glow:** Ambient diffused shadow `rgba(5, 150, 105, 0.20)` (cahaya Gaussian teredam lembut, meniadakan harsh neon bloom).
   - **Income / Surplus:** Emerald Soft Aksen (`#34d399`).
   - **Expense / Pengeluaran:** Soft Coral Rose (`#fb7185`).
   - **Transfer:** Muted Cyan Slate (`#38bdf8`).
   - **Budget Warning:** Soft Amber / Honey (`#fbbf24`).
   - **Budget Exceeded:** Controlled Crimson (`#f43f5e` dalam badge teredam, bukan banner mencolok).

---

## 4. Arsitektur Informasi & Struktur Halaman

### 4.1 Autentikasi (Authentication Flow)
1. **Sign In (Masuk):**
   - Header terverifikasi: Logo FISCA, badge *v2.4 Secure Node*, *End-to-End Encrypted*.
   - Input kredensial: Email/User ID & Password dengan toggle visibilitas dan *Ingat Saya*.
   - CTA utama: Tombol *Masuk ke FISCA* bergradasi emerald beludru.
   - Fast Auth: Opsi biometrik *Face ID & Sidik Jari* dan SSO (Google, Apple).
   - Trust badge: Jaminan privasi *Bank-Grade 256-Bit AES* (data terisolasi lokal, zero data selling).
2. **Sign Up (Pendaftaran):**
   - Input form lengkap: Nama Lengkap, Email Aktif, Kata Sandi dengan **Indikator Kekuatan Sandi** (*100% Kuat & Terisolasi*), dan Konfirmasi Sandi.
   - Persetujuan ketentuan & protokol *Zero-Knowledge Privacy*.
   - CTA pendaftaran: *Daftar Akun FISCA ⚡* dan divider SSO cepat.
   - Navigasi bolak-balik mulus ke Sign In.

### 4.2 Home (Financial Overview)
1. **Header:** Profil pengguna, sapaan personal ramah mata, indikator sinkronisasi cloud aman, dan lonceng notifikasi.
2. **Total Balance Glass Card:** Saldo bersih IDR dengan privasi toggle (sembunyikan angka), persentase pertumbuhan bulanan (+X.X%), dan indikator *Carbon-Aware Status*.
3. **Cash Flow Summary:** Kartu perbandingan Pemasukan (Income) dan Pengeluaran (Expense) berdampingan dalam material kaca beludru.
4. **Expense Breakdown:** Visualisasi distribusi kategori utama (*Kebutuhan Pokok, F&B, Transportasi, Langganan*).
5. **Recent Transactions:** Daftar riwayat transaksi terkini dengan ikon kategori terarah, nominal berwarna jelas, dan timestamp.
6. **Quick Voice Action:** Shortcut mengambang atau integrasi langsung ke Assistant.

### 4.3 History (Transaction Management)
1. **Pencarian Cerdas (Smart Search):** Input pencarian teks instan untuk deskripsi pedagang, kategori, atau tag.
2. **Multi-Dimensional Filter Chips:** Filter cepat Periode (*Bulan Ini, 7 Hari Terakhir, Kustom*), Jenis (*Semua, Pemasukan, Pengeluaran, Transfer*), dan Kategori.
3. **Ringkasan Filter Aktif:** Total volume transaksi sesuai kriteria yang disaring.
4. **Grouping per Tanggal:** Pemisahan riwayat transaksi berdasarkan hari (*Hari Ini, Kemarin, Tanggal Spesifik*) lengkap dengan akumulasi sub-total harian.
5. **Detail Transaksi & Undo:** Dukungan tombol *Batalkan (Undo)* untuk transaksi baru, serta opsi penyuntingan kategori dan akun.

### 4.4 FISCA Smart Assistant (Full-Screen AI Experience)
1. **State Transisi Layar Penuh:** Saat tombol Assistant di Bottom Navigation ditekan, antarmuka beralih ke mode layar penuh dan Bottom Navigation disembunyikan.
2. **Voice Interaction System:**
   - **Idle / Ready:** Siap mendengarkan, tombol mikrofon berdenyut lembut.
   - **Listening:** Audio waveform reaktif dengan pencahayaan ambient diffused.
   - **Processing / Transcribing:** Indikator pemrosesan pemahaman NLU.
   - **Confirmed / Saved:** Muncul kartu hasil pencatatan dengan tombol *Undo*.
3. **Rich Conversational Cards:**
   - *Transaction Result Card:* Rincian nominal, kategori terdeteksi, dompet/rekening, waktu, status validasi.
   - *Clarification Card:* Pertanyaan AI saat input ambigu (misal: "Apakah Rp 50.000 untuk Makan Siang atau Belanja Bahan Makanan?").
   - *Budget Proposal Card:* Perubahan alokasi anggaran yang diusulkan AI, menampilkan nilai sebelum/sesudah, dampak persentase, dan tombol *Setujui (Approve)* / *Tolak (Reject)*.
   - *Financial Insight & Largest Spending:* Jawaban analisis saat pengguna bertanya pengeluaran terbesar atau tren anomali.
4. **Chat Fallback:** Input field teks fleksibel bagi pengguna yang tidak ingin menggunakan suara.

### 4.5 Reports & Financial Analytics
1. **Period Switcher:** Pemilihan periode analisis (Bulanan, Kuartalan, Tahunan).
2. **Grafik Trend Arus Kas:** Visualisasi garis/batang yang responsif menampilkan rasio pemasukan vs pengeluaran.
3. **Kategori Pengeluaran Teratas:** Diagram komposisi alokasi dana dengan tooltip dan label nominal IDR.
4. **Insight Keuangan AI:** Ringkasan naratif otomatis mengenai efisiensi pengeluaran dan rasio tabungan.

### 4.6 Budget & Financial Planning
1. **Overall Budget Card:** Target total belanja bulanan, nominal terpakai, sisa saldo budget, dan progress bar bergradasi.
2. **Category Budget Items:**
   - *Normal (<80%):* Indikator emerald tenang.
   - *Warning (80–100%):* Indikator amber/kuning lembut.
   - *Over-Limit (>100%):* Badge peringatan merah bersahaja (*Warning Only*, pencatatan tetap berjalan).
3. **AI Allocation Recommendations:** Saran optimasi pemindahan alokasi budget antar-kategori untuk mencegah defisit.

---

## 5. Acceptance Criteria UI/UX

1. **Ergonomi Mata & Tampilan Layar:**
   - Seluruh latar belakang menggunakan tone obsidian gelap (`#0a0e16` – `#0f131c`).
   - Tidak ada elemen yang memancarkan silau neon menusuk (*harsh neon glare*); efek glow dibatasi pada difusi Gaussian lembut beropasitas rendah.
   - Semua teks utama menggunakan tone off-white (`#f1f5f9`), bukan putih murni (`#ffffff`).
2. **Pola Glassmorphism:**
   - Komponen kartu menggunakan `backdrop-filter: blur()`, background semi-transparan bergradasi, dan specular hairline border (`rgba(255,255,255,0.06)`).
3. **Hierarki Bottom Navigation:**
   - Terdapat tepat 5 menu: Home, History, FISCA Assistant, Reports, Profile.
   - Tombol tengah (FISCA Assistant) dibuat elevated dan menonjol.
   - Membuka Assistant wajib menyembunyikan navigasi bawah dan masuk ke mode layar penuh.
4. **Keamanan & Transparansi AI:**
   - Badge privasi dan enkripsi bank-grade tampak jelas di halaman autentikasi dan pengaturan akun.
   - AI hanya menandai transaksi "Berhasil Disimpan" setelah ada konfirmasi proses dan selalu menyediakan opsi pembatalan (*Undo*).

---

*Dokumen ini mengunci seluruh preferensi desain dan spesifikasi fungsional untuk implementasi lanjutan FISCA MVP.*
