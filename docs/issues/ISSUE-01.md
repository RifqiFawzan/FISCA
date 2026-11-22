# ISSUE-01 — Foundation (M1)

**Status:** CLOSED (QA PASS)

**Acceptance criteria**
1. `/health` mengembalikan `{status:"ok", db:"ok"}` termasuk ping database.
2. CORS hanya mengizinkan origin yang terdaftar; request `OPTIONS` ditangani.
3. Docker Compose menyalakan `db`, `db-init`, `api`, `web` dan `db-init` menjalankan goose `up` + seed (seed dilewati saat `APP_ENV=production`).
4. Kedua image multi-stage dan berjalan sebagai user non-root.
5. `go build ./...`, `go vet ./...`, `npm run lint`, `npm run build` hijau; mode host dev tetap tersedia.

**Implementasi:** `apps/api/cmd/{api,migrate,seed}`, `internal/server`, `internal/middleware`, `infra/docker-compose.yml`, `apps/{api,web}/Dockerfile`, `Taskfile.yml`, `package.json`.

**QA:** `docs/qa.md` §1 — health, CORS, status container, build/test bersih. Dikerjakan sebelum sesi implementasi M2–M7, sehingga sebelumnya belum tercatat sebagai issue; ditambahkan di sini agar penomoran tidak berlubang. → **PASS**
