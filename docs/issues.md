# FISCA — Issues

Format mengikuti `workflow.md`: setiap issue punya acceptance criteria, lalu verifikasi QA independen sebelum ditutup.

**Penomoran:** `ISSUE-01` s.d. `ISSUE-26` — **25 CLOSED (QA PASS)** dan **1 belum dikerjakan** (0 `OPEN`, 1 `BLOCKED`; rincian di tabel bawah).
**Nomor berikutnya yang belum dipakai: `ISSUE-27`.** Jangan memakai ulang nomor lama saat menambah issue baru; lanjutkan dari 27.

Status:
- `CLOSED (QA PASS)` = acceptance criteria sudah diverifikasi QA.
- `OPEN` = belum dikerjakan, siap jadi bahan kerja berikutnya.
- `BLOCKED` = siap dikerjakan tetapi menunggu kredensial/parameter dari luar.

| Issue | Isi | Status | Berkas |
|---|---|---|---|
| ISSUE-01 | M1 Foundation | CLOSED (QA PASS) | [ISSUE-01.md](./issues/ISSUE-01.md) |
| ISSUE-02 | M2 Ledger & data master | CLOSED (QA PASS) | [ISSUE-02.md](./issues/ISSUE-02.md) |
| ISSUE-03 | M3 Transaksi & idempotency | CLOSED (QA PASS) | [ISSUE-03.md](./issues/ISSUE-03.md) |
| ISSUE-04 | M4 Dashboard & analitik | CLOSED (QA PASS) | [ISSUE-04.md](./issues/ISSUE-04.md) |
| ISSUE-05 | M5 Historical intelligence | CLOSED (QA PASS) | [ISSUE-05.md](./issues/ISSUE-05.md) |
| ISSUE-06 | M6 Budget & advisor | CLOSED (QA PASS) | [ISSUE-06.md](./issues/ISSUE-06.md) |
| ISSUE-07 | M7 Frontend, hardening & observability | CLOSED (QA PASS) | [ISSUE-07.md](./issues/ISSUE-07.md) |
| ISSUE-08 | Klarifikasi draft (E2E #3) | CLOSED (QA PASS) | [ISSUE-08.md](./issues/ISSUE-08.md) |
| ISSUE-09 | Tanggal relatif di asisten ("kemarin", "tgl 15") | CLOSED (QA PASS) | [ISSUE-09.md](./issues/ISSUE-09.md) |
| ISSUE-10 | Integration & security test otomatis (PRD exit #19, #20) | CLOSED (QA PASS) | [ISSUE-10.md](./issues/ISSUE-10.md) |
| ISSUE-11 | Login/register, mode JWT, logout (PRD exit #1) | CLOSED (QA PASS) | [ISSUE-11.md](./issues/ISSUE-11.md) |
| ISSUE-12 | Multi-transaksi satu kalimat (PRD §22.3) | CLOSED (QA PASS) | [ISSUE-12.md](./issues/ISSUE-12.md) |
| ISSUE-13 | Koreksi transaksi lewat chat ("ubah kopi tadi…") | CLOSED (QA PASS) | [ISSUE-13.md](./issues/ISSUE-13.md) |
| ISSUE-14 | Tanggal & frasa lanjutan ("minggu lalu", "tgl 15 Oktober") | CLOSED (QA PASS) | [ISSUE-14.md](./issues/ISSUE-14.md) |
| ISSUE-15 | Auth lokal: email + kata sandi (pengganti jalur Supabase) | CLOSED (QA PASS) | [ISSUE-15.md](./issues/ISSUE-15.md) |
| ISSUE-16 | Verifikasi voice/STT end-to-end | BLOCKED (butuh `OPENAI_API_KEY`) | [ISSUE-16.md](./issues/ISSUE-16.md) |
| ISSUE-17 | Eval set AI 40 kasus Bahasa Indonesia | CLOSED (QA PASS) | [ISSUE-17.md](./issues/ISSUE-17.md) |
| ISSUE-18 | Playwright E2E wajib (plan §12.5) | CLOSED (QA PASS) | [ISSUE-18.md](./issues/ISSUE-18.md) |
| ISSUE-19 | OpenAPI + codegen DTO TypeScript | CLOSED (QA PASS) | [ISSUE-19.md](./issues/ISSUE-19.md) |
| ISSUE-20 | Savings Planner & Purchase Advisor (PRD §4.2) | CLOSED (QA PASS) | [ISSUE-20.md](./issues/ISSUE-20.md) |
| ISSUE-21 | Audit a11y & responsivitas UI | CLOSED (QA PASS) | [ISSUE-21.md](./issues/ISSUE-21.md) |
| ISSUE-22 | Form masuk & daftar: mata password, konfirmasi sandi, nama depan/belakang | CLOSED (QA PASS) | [ISSUE-22.md](./issues/ISSUE-22.md) |
| ISSUE-23 | Penyelarasan design Dark Carbon Glass: warna, badge, distribusi, tile | CLOSED (QA PASS) | [ISSUE-23.md](./issues/ISSUE-23.md) |
| ISSUE-24 | Reset password lokal (CLI) | CLOSED (QA PASS) | [ISSUE-24.md](./issues/ISSUE-24.md) |
| ISSUE-25 | Cakupan verifikasi: typecheck test + e2e tiap halaman + celah tercatat | CLOSED (QA PASS) | [ISSUE-25.md](./issues/ISSUE-25.md) |
| ISSUE-26 | Kurangi dosis glass, glow, dan ikon sesuai aturan antislop | CLOSED (QA PASS) | [ISSUE-26.md](./issues/ISSUE-26.md) |

---

> **Satu issue per berkas** di folder docs/issues/. Berkas di atas adalah indeks:
> penomoran, legenda status, dan tabel lengkap tetap ada di sini.

---

# Belum dikerjakan (terblokir)

- **ISSUE-16 — Verifikasi voice/STT** berstatus `BLOCKED`: menunggu `OPENAI_API_KEY` atau provider setara. Ini **satu-satunya issue yang tersisa**.

Urutan pengerjaan mengikuti PRD §23 (exit criteria) dan plan.md milestone.

---
