-- name: CreateAccount :one
INSERT INTO accounts (user_id, name, type, opening_balance, is_default)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListAccountsByUser :many
SELECT *
FROM accounts
WHERE user_id = $1
  AND archived_at IS NULL
ORDER BY is_default DESC, name ASC;

-- name: GetAccountForUser :one
SELECT *
FROM accounts
WHERE id = $1
  AND user_id = $2;

-- name: GetDefaultAccount :one
SELECT *
FROM accounts
WHERE user_id = $1
  AND is_default
  AND archived_at IS NULL
LIMIT 1;

-- name: CountActiveAccounts :one
SELECT COUNT(*)::bigint
FROM accounts
WHERE user_id = $1
  AND archived_at IS NULL;

-- name: UpdateAccount :one
UPDATE accounts
SET name = COALESCE(sqlc.narg('name'), name),
    type = COALESCE(sqlc.narg('type'), type)
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND archived_at IS NULL
RETURNING *;

-- name: ClearDefaultAccount :exec
UPDATE accounts
SET is_default = false
WHERE user_id = $1
  AND is_default;

-- name: SetDefaultAccount :exec
UPDATE accounts
SET is_default = true
WHERE id = $1
  AND user_id = $2
  AND archived_at IS NULL;

-- name: ArchiveAccount :exec
UPDATE accounts
SET archived_at = now(),
    is_default  = false
WHERE id = $1
  AND user_id = $2;

-- name: ListAccountBalances :many
SELECT
    a.id,
    a.name,
    a.type,
    a.is_default,
    a.opening_balance,
    a.created_at,
    (
        a.opening_balance
        + COALESCE(SUM(CASE WHEN t.type = 'income' AND t.account_id = a.id THEN t.amount END), 0)
        - COALESCE(SUM(CASE WHEN t.type = 'expense' AND t.account_id = a.id THEN t.amount END), 0)
        + COALESCE(SUM(CASE WHEN t.type = 'transfer' AND t.to_account_id = a.id THEN t.amount END), 0)
        - COALESCE(SUM(CASE WHEN t.type = 'transfer' AND t.from_account_id = a.id THEN t.amount END), 0)
    )::bigint AS balance
FROM accounts a
LEFT JOIN transactions t
    ON t.user_id = a.user_id
   AND t.status = 'active'
   AND (t.account_id = a.id OR t.from_account_id = a.id OR t.to_account_id = a.id)
WHERE a.user_id = $1
  AND a.archived_at IS NULL
GROUP BY a.id, a.name, a.type, a.is_default, a.opening_balance, a.created_at
ORDER BY a.is_default DESC, a.name ASC;
