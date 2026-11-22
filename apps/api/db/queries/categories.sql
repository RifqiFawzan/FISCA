-- name: CreateCategory :one
INSERT INTO categories (user_id, name, kind, icon, color, is_default)
VALUES ($1, $2, $3, $4, $5, false)
RETURNING *;

-- name: ListCategoriesByUser :many
SELECT *
FROM categories
WHERE user_id = $1
  AND archived_at IS NULL
  AND (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind'))
ORDER BY kind ASC, name ASC;

-- name: GetCategoryForUser :one
SELECT *
FROM categories
WHERE id = $1
  AND user_id = $2;

-- name: UpdateCategory :one
UPDATE categories
SET name  = COALESCE(sqlc.narg('name'), name),
    icon  = COALESCE(sqlc.narg('icon'), icon),
    color = COALESCE(sqlc.narg('color'), color)
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND archived_at IS NULL
RETURNING *;

-- name: SeedDefaultCategories :exec
INSERT INTO categories (user_id, name, kind, icon, color, is_default)
SELECT sqlc.arg('user_id'), x.name, x.kind, x.icon, x.color, true
FROM jsonb_to_recordset(sqlc.arg('items')::jsonb)
     AS x(name text, kind text, icon text, color text)
ON CONFLICT DO NOTHING;

-- name: ArchiveCategory :exec
UPDATE categories
SET archived_at = now()
WHERE id = $1
  AND user_id = $2;
