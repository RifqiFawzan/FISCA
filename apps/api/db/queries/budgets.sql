-- ---------------------------------------------------------------------------
-- budgets & budget categories
-- ---------------------------------------------------------------------------

-- name: CreateBudget :one
INSERT INTO budgets (user_id, period_start, period_end, overall_limit)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetBudgetForUser :one
SELECT *
FROM budgets
WHERE id = $1
  AND user_id = $2;

-- name: GetBudgetForPeriod :one
SELECT *
FROM budgets
WHERE user_id = $1
  AND period_start = $2
  AND status = 'active'
LIMIT 1;

-- name: ListBudgetsByUser :many
SELECT *
FROM budgets
WHERE user_id = $1
  AND status = 'active'
ORDER BY period_start DESC;

-- name: UpdateBudget :one
UPDATE budgets
SET overall_limit = COALESCE(sqlc.narg('overall_limit'), overall_limit),
    period_end    = COALESCE(sqlc.narg('period_end'), period_end),
    version       = version + 1
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND version = sqlc.arg('expected_version')
  AND status = 'active'
RETURNING *;

-- name: ArchiveBudget :exec
UPDATE budgets
SET status  = 'archived',
    version = version + 1
WHERE id = $1
  AND user_id = $2
  AND status = 'active';

-- name: UpsertBudgetCategory :one
INSERT INTO budget_categories (budget_id, category_id, limit_amount)
VALUES ($1, $2, $3)
ON CONFLICT (budget_id, category_id) DO UPDATE
    SET limit_amount = EXCLUDED.limit_amount,
        updated_at   = now()
RETURNING *;

-- name: DeleteBudgetCategory :exec
DELETE FROM budget_categories
WHERE budget_id = $1
  AND category_id = $2;

-- name: DeleteAllBudgetCategories :exec
DELETE FROM budget_categories
WHERE budget_id = $1;

-- name: LockBudget :one
SELECT *
FROM budgets
WHERE id = $1
  AND user_id = $2
  AND status = 'active'
FOR UPDATE;

-- name: ListBudgetCategories :many
SELECT *
FROM budget_categories
WHERE budget_id = $1
ORDER BY limit_amount DESC;

-- ---------------------------------------------------------------------------
-- budget proposals
-- ---------------------------------------------------------------------------

-- name: CreateBudgetProposal :one
INSERT INTO budget_proposals (user_id, budget_id, base_version, changes, rationale, impact)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetBudgetProposalForUser :one
SELECT *
FROM budget_proposals
WHERE id = $1
  AND user_id = $2;

-- name: ListBudgetProposals :many
SELECT *
FROM budget_proposals
WHERE user_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT sqlc.arg('page_size')::int;

-- name: UpdateBudgetProposalChanges :one
UPDATE budget_proposals
SET changes = sqlc.arg('changes'),
    rationale = COALESCE(sqlc.narg('rationale'), rationale)
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND status = 'pending'
RETURNING *;

-- name: DecideBudgetProposal :one
UPDATE budget_proposals
SET status     = sqlc.arg('status'),
    decided_at = now()
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND status = 'pending'
RETURNING *;

-- name: LockBudgetProposal :one
SELECT *
FROM budget_proposals
WHERE id = $1
  AND user_id = $2
FOR UPDATE;

-- name: ExpireStaleBudgetProposals :exec
UPDATE budget_proposals
SET status = 'expired'
WHERE user_id = $1
  AND status = 'pending'
  AND expires_at <= now();
