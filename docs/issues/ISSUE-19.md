# ISSUE-19 — OpenAPI + codegen DTO TypeScript

**Status:** CLOSED (QA PASS)

**Sumber:** plan.md §12.3.

**Acceptance criteria**
1. `docs/api/openapi.yaml` mencerminkan semua endpoint, skema respons, dan kode error yang ada di `internal/api`.
2. Tipe TypeScript di `lib/api.ts` dihasilkan dari OpenAPI (bukan ditulis tangan) atau diverifikasi otomatis.
3. Test kontrak memastikan `card.type` dan bentuk dashboard tidak berubah tanpa perubahan versi.
4. Perubahan DTO tanpa memperbarui kontrak membuat test gagal.

**Implementasi**
- `docs/api/openapi.yaml`: **28 path** (semua endpoint `server.New` + `/health`), 60+ skema, envelope sukses/gagal, parameter bersama, request body, serta enum `ErrorCode` (10 kode, plan.md §7.1) dan `AssistantCardType` (10 kartu, plan.md §7.3).
- `npm run gen:api` → `apps/web/lib/api.gen.ts` (openapi-typescript 7.13, di-commit seperti output sqlc). Tipe DTO di `lib/api.ts` kini **alias ke kontrak**, jadi tidak ada lagi definisi wire yang ditulis dua kali; hanya normalisasi klien (`nextCursor: null`) yang tetap tangan.
- `internal/server/contract_test.go`: validator JSON-Schema subset (`$ref`, `oneOf`, `type`, `properties`, `required`, `items`, `enum`, `nullable`, `additionalProperties`) yang memvalidasi respons live terhadap YAML. Ia memicu **semua 28 path**, memaksa status yang terdokumentasi, mengumpulkan `card.type` dan `error.code`, dan gagal bila ada operasi kontrak yang tidak pernah dipanggil.
- Dependensi baru: `gopkg.in/yaml.v3` (Go) dan `openapi-typescript` (devDependency npm).

**QA**
- `go test ./internal/server/ -run Contract -v` → **PASS**: 28 paths, 9 kartu teramati, 3 kode error teramati.
- `go vet ./...`, `go test ./... -count=1` (dengan `TEST_DATABASE_URL`) → bersih.
- `npm run gen:api` exit 0; `tsc --noEmit`, `npm run lint`, `npm run build` bersih; `npm run stack:up` + `npm run smoke:api` lulus; `npm run test:e2e` **24/24 lulus**.

**Temuan QA: kontrak saya yang salah, ditangkap test (bukan sebaliknya)**
1. `TransactionDTO` dan `CreateTransactionInput` **tidak** memakai `omitempty`, jadi field pointer selalu terkirim sebagai `null`. Skema awal saya menganggapnya opsional → validator melaporkan 13 field.
2. `card.draft` memakai struct request yang selalu mengirim `null` untuk field kosong → dipisah menjadi skema `TransactionDraft` (request dan response memang butuh `required` berbeda).
3. `BudgetProposal.status` enum kekurangan `stale`; `Transaction.source` seharusnya `[manual, voice, chat]` (bukan memuat `ai`).
4. Kebalikannya juga benar dan diverifikasi: `AssistantMessage.card` memang `omitempty` (opsional), sedangkan `rationale`, `decidedAt`, dan `title` selalu ikut terkirim.
5. Bug di test sendiri: `operation()` membandingkan `"get"` dengan konstanta `"GET"` sehingga semua operasi dilaporkan tidak ada.

**Batasan:** test memvalidasi **respons**, bukan request body; kartu `error` terdokumentasi tetapi belum bisa dipicu oleh jalur deterministik (butuh penolakan service yang bisa ditiru), jadi hanya divalidasi lewat enum; belum ada CI yang menjalankan test kontrak otomatis.
