-- +goose Up

-- ---------------------------------------------------------------------------
-- auth lokal (ISSUE-15)
--
-- FISCA dipakai untuk pemakaian pribadi lebih dulu (D-01 APPROVED), jadi
-- autentikasi dijalankan sendiri tanpa pihak ketiga:
--
--   * users.password_hash  Argon2id dari kata sandi. Nullable karena baris yang
--                           sudah ada (dan user yang lahir lewat jalur JWT yang
--                           masih dorman) tidak punya kata sandi.
--   * sessions             Token sesi disimpan *ter-hash* (SHA-256), bukan
--                           mentah. Argon2id sengaja TIDAK dipakai di sini:
--                           token acak 32 byte sudah berentropi tinggi dan
--                           tabel ini dibaca pada setiap request — Argon2id
--                           memang dibuat lambat (~50ms) dan akan melambatkan
--                           seluruh API. Sama prinsipnya dengan password:
--                           yang lambat hanya untuk tebakan yang mudah ditebak.
--
-- AUTH_MODE=dev dihapus pada issue ini: satu jalur auth, tanpa token
-- `dev:<uuid>` yang dipercaya apa adanya.
-- ---------------------------------------------------------------------------

ALTER TABLE users
    ADD COLUMN password_hash TEXT;

CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_idx
    ON sessions (user_id)
    WHERE revoked_at IS NULL;

-- Untuk pembersihan sesi kedaluwarsa secara berkala.
CREATE INDEX sessions_expires_idx
    ON sessions (expires_at);

-- +goose Down

DROP TABLE IF EXISTS sessions;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
