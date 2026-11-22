-- name: UpsertUser :one
INSERT INTO users (id, email, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
    SET email        = EXCLUDED.email,
        display_name = EXCLUDED.display_name,
        updated_at   = now()
RETURNING *;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;

-- name: UpdateUserPreferences :one
UPDATE users
SET display_name = COALESCE(sqlc.narg('display_name'), display_name),
    timezone     = COALESCE(sqlc.narg('timezone'), timezone),
    locale       = COALESCE(sqlc.narg('locale'), locale)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: MarkUserOnboarded :exec
UPDATE users
SET onboarded_at = COALESCE(onboarded_at, now())
WHERE id = $1;
