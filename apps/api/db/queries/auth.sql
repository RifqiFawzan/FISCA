-- Auth lokal (ISSUE-15). Hash kata sandi dihitung di Go (Argon2id); di sini
-- hanya disimpan hasilnya. Token sesi disimpan sebagai SHA-256 sehingga token
-- mentah tidak pernah menyentuh disk.

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = sqlc.arg('email');

-- name: CreateUserWithPassword :one
INSERT INTO users (id, email, display_name, password_hash)
VALUES (sqlc.arg('id'), sqlc.arg('email'), sqlc.arg('display_name'), sqlc.arg('password_hash'))
RETURNING *;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES (sqlc.arg('user_id'), sqlc.arg('token_hash'), sqlc.arg('expires_at'))
RETURNING *;

-- name: GetActiveSessionByTokenHash :one
-- Sesi yang sudah dicabut atau lewat masa berlaku dianggap tidak ada, jadi
-- verifier tidak perlu memeriksa tiga kondisi secara terpisah.
SELECT s.user_id, s.expires_at
FROM sessions s
WHERE s.token_hash = sqlc.arg('token_hash')
  AND s.revoked_at IS NULL
  AND s.expires_at > now();

-- name: RevokeSessionByTokenHash :execrows
UPDATE sessions
SET revoked_at = now()
WHERE token_hash = sqlc.arg('token_hash')
  AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at < now();

-- name: UpdateUserPassword :one
-- Dipakai reset password lokal (cmd/resetpassword). Mengembalikan baris agar
-- pemanggil bisa memastikan email benar-benar ada sebelum mencabut sesi.
UPDATE users
SET password_hash = sqlc.arg('password_hash')
WHERE email = sqlc.arg('email')
RETURNING *;

-- name: RevokeAllSessionsForUser :execrows
-- Reset password harus mematikan SEMUA sesi lama, bukan hanya yang sedang
-- dipakai: pemulihan biasanya terjadi karena akun dianggap tidak aman lagi.
UPDATE sessions
SET revoked_at = now()
WHERE user_id = sqlc.arg('user_id')
  AND revoked_at IS NULL;
