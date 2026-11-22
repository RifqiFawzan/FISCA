-- ---------------------------------------------------------------------------
-- transactions
-- ---------------------------------------------------------------------------

-- name: CreateTransaction :one
INSERT INTO transactions (
    user_id,
    type,
    amount,
    description,
    category_id,
    account_id,
    from_account_id,
    to_account_id,
    payment_method,
    merchant_name,
    occurred_at,
    notes,
    source
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: GetTransactionForUser :one
SELECT *
FROM transactions
WHERE id = $1
  AND user_id = $2;

-- name: ListTransactions :many
SELECT *
FROM transactions
WHERE user_id = sqlc.arg('user_id')
  AND status = 'active'
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id'))
  AND (
        sqlc.narg('account_id')::uuid IS NULL
        OR account_id = sqlc.narg('account_id')
        OR from_account_id = sqlc.narg('account_id')
        OR to_account_id = sqlc.narg('account_id')
      )
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR occurred_at >= sqlc.narg('from_ts'))
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR occurred_at < sqlc.narg('to_ts'))
  AND (sqlc.narg('min_amount')::bigint IS NULL OR amount >= sqlc.narg('min_amount'))
  AND (sqlc.narg('max_amount')::bigint IS NULL OR amount <= sqlc.narg('max_amount'))
  AND (
        sqlc.narg('cursor_ts')::timestamptz IS NULL
        OR occurred_at < sqlc.narg('cursor_ts')
        OR (occurred_at = sqlc.narg('cursor_ts') AND id < sqlc.narg('cursor_id'))
      )
  AND (
        sqlc.narg('q')::text IS NULL
        OR description ILIKE '%' || sqlc.narg('q') || '%'
        OR merchant_name ILIKE '%' || sqlc.narg('q') || '%'
      )
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg('page_size')::int;

-- name: ListTransactionsWithAudit :many
SELECT
    t.*,
    al.action        AS audit_action,
    al.actor         AS audit_actor,
    al.created_at    AS audit_created_at
FROM transactions t
LEFT JOIN audit_logs al
    ON al.entity_type = 'transaction'
   AND al.entity_id = t.id
   AND al.user_id = t.user_id
WHERE t.id = sqlc.arg('id')
  AND t.user_id = sqlc.arg('user_id')
ORDER BY al.created_at DESC;

-- name: UpdateTransaction :one
UPDATE transactions
SET description = COALESCE(sqlc.narg('description'), description),
    amount      = COALESCE(sqlc.narg('amount'), amount),
    category_id = COALESCE(sqlc.narg('category_id'), category_id),
    occurred_at = COALESCE(sqlc.narg('occurred_at'), occurred_at),
    notes       = COALESCE(sqlc.narg('notes'), notes),
    version     = version + 1
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND version = sqlc.arg('expected_version')
  AND status = 'active'
RETURNING *;

-- name: VoidTransaction :one
UPDATE transactions
SET status      = 'voided',
    voided_at   = now(),
    void_reason = sqlc.arg('reason'),
    version     = version + 1
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND status = 'active'
RETURNING *;
