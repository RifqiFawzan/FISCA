-- ---------------------------------------------------------------------------
-- analytics (semua agregasi uang dilakukan di backend, plan.md D-11)
-- ---------------------------------------------------------------------------

-- name: GetPeriodTotals :one
SELECT
    COALESCE(SUM(CASE WHEN type = 'income' THEN amount END), 0)::bigint  AS income,
    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount END), 0)::bigint AS expense,
    COALESCE(SUM(CASE WHEN type = 'income' THEN amount
                      WHEN type = 'expense' THEN -amount END), 0)::bigint AS net
FROM transactions
WHERE user_id = sqlc.arg('user_id')
  AND status = 'active'
  AND occurred_at >= sqlc.arg('from_ts')
  AND occurred_at < sqlc.arg('to_ts');

-- name: GetSpendingByCategory :many
SELECT
    t.category_id,
    COALESCE(c.name, 'Uncategorized') AS category_name,
    COALESCE(c.color, '#64748B')      AS color,
    SUM(t.amount)::bigint             AS total,
    COUNT(*)::bigint                  AS transaction_count
FROM transactions t
LEFT JOIN categories c
    ON c.id = t.category_id
   AND c.user_id = t.user_id
WHERE t.user_id = sqlc.arg('user_id')
  AND t.status = 'active'
  AND t.type = 'expense'
  AND t.occurred_at >= sqlc.arg('from_ts')
  AND t.occurred_at < sqlc.arg('to_ts')
GROUP BY t.category_id, c.name, c.color
ORDER BY total DESC;

-- name: GetCashFlowBuckets :many
SELECT
    date_trunc(sqlc.arg('bucket')::text, t.occurred_at)::timestamptz AS bucket_start,
    COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount END), 0)::bigint  AS income,
    COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount END), 0)::bigint AS expense
FROM transactions t
WHERE t.user_id = sqlc.arg('user_id')
  AND t.status = 'active'
  AND t.type IN ('income', 'expense')
  AND t.occurred_at >= sqlc.arg('from_ts')
  AND t.occurred_at < sqlc.arg('to_ts')
GROUP BY 1
ORDER BY 1;

-- name: GetRecentTransactions :many
SELECT *
FROM transactions
WHERE user_id = $1
  AND status = 'active'
ORDER BY occurred_at DESC, created_at DESC
LIMIT $2;

-- name: GetLargestTransactions :many
SELECT *
FROM transactions
WHERE user_id = sqlc.arg('user_id')
  AND status = 'active'
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id'))
  AND occurred_at >= sqlc.arg('from_ts')
  AND occurred_at < sqlc.arg('to_ts')
  AND (sqlc.narg('min_amount')::bigint IS NULL OR amount >= sqlc.narg('min_amount'))
  AND (sqlc.narg('max_amount')::bigint IS NULL OR amount <= sqlc.narg('max_amount'))
ORDER BY amount DESC, occurred_at DESC
LIMIT sqlc.arg('page_size')::int;

-- name: GetBudgetSpend :one
SELECT
    b.id,
    b.period_start,
    b.period_end,
    b.overall_limit,
    b.version,
    COALESCE(SUM(t.amount), 0)::bigint AS spent
FROM budgets b
LEFT JOIN transactions t
    ON t.user_id = b.user_id
   AND t.status = 'active'
   AND t.type = 'expense'
   AND t.occurred_at >= b.period_start::timestamptz
   AND t.occurred_at < b.period_end::timestamptz
WHERE b.id = sqlc.arg('budget_id')
  AND b.user_id = sqlc.arg('user_id')
GROUP BY b.id;

-- name: GetCategoryBudgetSpend :many
SELECT
    bc.category_id,
    COALESCE(c.name, 'Uncategorized') AS category_name,
    bc.limit_amount,
    COALESCE(SUM(t.amount), 0)::bigint AS spent
FROM budget_categories bc
JOIN budgets b
    ON b.id = bc.budget_id
LEFT JOIN categories c
    ON c.id = bc.category_id
LEFT JOIN transactions t
    ON t.user_id = b.user_id
   AND t.category_id = bc.category_id
   AND t.status = 'active'
   AND t.type = 'expense'
   AND t.occurred_at >= b.period_start::timestamptz
   AND t.occurred_at < b.period_end::timestamptz
WHERE bc.budget_id = sqlc.arg('budget_id')
  AND b.user_id = sqlc.arg('user_id')
GROUP BY bc.id, bc.category_id, c.name, bc.limit_amount
ORDER BY c.name ASC;

-- name: GetMonthlyCategoryTotals :many
SELECT
    date_trunc('month', t.occurred_at)::date AS month,
    t.category_id,
    COALESCE(c.name, 'Uncategorized')       AS category_name,
    SUM(t.amount)::bigint                   AS total,
    COUNT(*)::bigint                        AS transaction_count
FROM transactions t
LEFT JOIN categories c
    ON c.id = t.category_id
WHERE t.user_id = sqlc.arg('user_id')
  AND t.status = 'active'
  AND t.type = 'expense'
  AND t.occurred_at >= sqlc.arg('from_ts')
  AND t.occurred_at < sqlc.arg('to_ts')
  AND (sqlc.narg('category_id')::uuid IS NULL OR t.category_id = sqlc.narg('category_id'))
GROUP BY 1, 2, 3
ORDER BY 1 ASC;

-- name: GetMonthlyTotals :many
SELECT
    date_trunc('month', occurred_at)::date AS month,
    COALESCE(SUM(CASE WHEN type = 'income' THEN amount END), 0)::bigint  AS income,
    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount END), 0)::bigint AS expense
FROM transactions
WHERE user_id = sqlc.arg('user_id')
  AND status = 'active'
  AND type IN ('income', 'expense')
  AND occurred_at >= sqlc.arg('from_ts')
  AND occurred_at < sqlc.arg('to_ts')
GROUP BY 1
ORDER BY 1 ASC;
