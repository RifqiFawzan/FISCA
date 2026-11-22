-- ---------------------------------------------------------------------------
-- audit log (plan.md §5.3)
-- ---------------------------------------------------------------------------

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
    user_id, actor, entity_type, entity_id, action, before, after, request_id
) VALUES (
    sqlc.arg('user_id'),
    sqlc.arg('actor'),
    sqlc.arg('entity_type'),
    sqlc.narg('entity_id'),
    sqlc.arg('action'),
    sqlc.narg('before'),
    sqlc.narg('after'),
    sqlc.narg('request_id')
);

-- name: ListAuditLogsForEntity :many
SELECT *
FROM audit_logs
WHERE user_id = sqlc.arg('user_id')
  AND entity_type = sqlc.arg('entity_type')
  AND entity_id = sqlc.arg('entity_id')
ORDER BY created_at DESC
LIMIT sqlc.arg('page_size')::int;
