# ISSUE-07 — Frontend, hardening & observability (M7)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. Halaman `/home`, `/history`, `/reports`, `/budget`, `/profile`, `/assistant` terhubung ke API nyata dengan state loading/error/kosong.
2. Mobile-first: tidak ada horizontal overflow pada viewport 360px; bottom nav 5 item; asisten workspace terpisah.
3. Rate limit mengembalikan `429 RATE_LIMITED` + `Retry-After`; AI/STT punya budget lebih ketat.
4. Log operasional JSON memuat `requestId`, `route`, `status`, `latencyMs`.
5. Tidak ada copy/UI carbon, ESG, atau clean-energy; tidak ada klaim keamanan yang belum diimplementasi.

**Implementasi:** `apps/web/lib/{api,format,use-api}.ts`, `components/ui/*`, `components/views/*`, `internal/middleware/ratelimit.go`, `cmd/api/main.go`.

**QA:** `npm run lint`, `tsc --noEmit`, `npm run build`; CDP viewport 360px → `scrollWidth=360` di 6 halaman; screenshot desktop 1280px; `curl.exe` 30 request cepat → 6× `429`; `docker logs fisca-api` → baris JSON `WARN http_request`. E2E browser: create "17rb" → notifikasi sukses + saldo berubah, asisten summary card, riwayat search natural, budget approve. → **PASS**
