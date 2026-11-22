-- ---------------------------------------------------------------------------
-- savings goals (plan.md §7.2 tambahan, ISSUE-20)
--
-- Progress is never stored: it is derived from the linked account's ledger
-- balance at read time, which is what keeps the money counted only once.
-- ---------------------------------------------------------------------------

-- name: CreateSavingsGoal :one
INSERT INTO savings_goals (user_id, name, target_amount, deadline, account_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSavingsGoalForUser :one
SELECT *
FROM savings_goals
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND archived_at IS NULL;

-- name: ListSavingsGoalsByUser :many
SELECT *
FROM savings_goals
WHERE user_id = sqlc.arg('user_id')
  AND archived_at IS NULL
ORDER BY deadline ASC, created_at ASC;

-- name: UpdateSavingsGoal :one
UPDATE savings_goals
SET name          = sqlc.arg('name'),
    target_amount = sqlc.arg('target_amount'),
    deadline      = sqlc.arg('deadline'),
    account_id    = sqlc.arg('account_id'),
    version       = version + 1
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND version = sqlc.arg('expected_version')
  AND archived_at IS NULL
RETURNING *;

-- name: ArchiveSavingsGoal :exec
UPDATE savings_goals
SET archived_at = now()
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND archived_at IS NULL;

-- ---------------------------------------------------------------------------
-- purchase affordability (PRD §10.6): every number comes from the ledger.
-- ---------------------------------------------------------------------------

-- name: CountActiveMonths :one
SELECT COUNT(DISTINCT date_trunc('month', occurred_at))::int AS months
FROM transactions
WHERE user_id = sqlc.arg('user_id')
  AND status = 'active'
  AND occurred_at >= sqlc.arg('from_ts')
  AND occurred_at < sqlc.arg('to_ts');
