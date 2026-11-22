-- ---------------------------------------------------------------------------
-- idempotency (plan.md §5.3, D-07)
--
-- Reservation model: a row is inserted *before* the work runs so two
-- concurrent requests with the same key cannot both execute. status_code = 0
-- means "in progress"; the handler completes it once the response is known.
-- ---------------------------------------------------------------------------

-- name: GetIdempotencyKey :one
SELECT *
FROM idempotency_keys
WHERE user_id = $1
  AND key = $2
  AND endpoint = $3;

-- name: ReserveIdempotencyKey :execrows
INSERT INTO idempotency_keys (user_id, key, endpoint, request_hash, status_code, response_body)
VALUES ($1, $2, $3, $4, 0, NULL)
ON CONFLICT (user_id, key, endpoint) DO NOTHING;

-- name: CompleteIdempotencyKey :exec
UPDATE idempotency_keys
SET status_code   = sqlc.arg('status_code'),
    response_body = sqlc.arg('response_body')
WHERE user_id = sqlc.arg('user_id')
  AND key = sqlc.arg('key')
  AND endpoint = sqlc.arg('endpoint')
  AND status_code = 0;

-- name: DeleteIdempotencyKey :exec
DELETE FROM idempotency_keys
WHERE user_id = $1
  AND key = $2
  AND endpoint = $3
  AND status_code = 0;

-- name: DeleteExpiredIdempotencyKeys :exec
DELETE FROM idempotency_keys
WHERE created_at < now() - INTERVAL '7 days';
