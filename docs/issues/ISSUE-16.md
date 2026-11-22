# ISSUE-16 — Verifikasi voice/STT end-to-end

**Status:** BLOCKED — butuh `OPENAI_API_KEY` (atau provider setara) dan `OPENAI_STT_MODEL`.

**Acceptance criteria**
1. Rekaman suara → transkrip benar untuk Bahasa Indonesia informal.
2. Transkrip masuk ke jalur yang sama dengan chat (parse → validasi → kartu).
3. Izin mikrofon ditolak → jatuh ke chat tanpa hilangnya teks yang sudah diketik (PRD exit #18).
4. Tanpa key → `503 AI_UNAVAILABLE` dengan pesan yang bisa ditindaklanjuti (sudah teruji sebagian di `scripts/smoke.ps1`).
5. Transkrip ditampilkan sebelum dieksekusi bila keyakinan rendah (plan.md mitigasi risiko STT).

**Rencana:** backend sudah siap (`POST /speech/transcriptions`); yang kurang bukti live + UI fallback izin.
