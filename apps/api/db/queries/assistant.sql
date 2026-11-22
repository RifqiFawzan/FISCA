-- ---------------------------------------------------------------------------
-- assistant conversations & messages (plan.md §5.3, §9)
-- ---------------------------------------------------------------------------

-- name: CreateConversation :one
INSERT INTO assistant_conversations (user_id, title)
VALUES ($1, $2)
RETURNING *;

-- name: GetConversationForUser :one
SELECT *
FROM assistant_conversations
WHERE id = $1
  AND user_id = $2;

-- name: ListConversations :many
SELECT *
FROM assistant_conversations
WHERE user_id = $1
ORDER BY last_message_at DESC
LIMIT $2;

-- name: TouchConversation :exec
UPDATE assistant_conversations
SET last_message_at = now(),
    title = COALESCE(sqlc.narg('title'), title)
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id');

-- name: DeleteConversation :exec
DELETE FROM assistant_conversations
WHERE id = $1
  AND user_id = $2;

-- name: CreateAssistantMessage :one
INSERT INTO assistant_messages (
    conversation_id, user_id, role, content, tool_calls, card
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListAssistantMessages :many
SELECT *
FROM assistant_messages
WHERE conversation_id = $1
  AND user_id = $2
ORDER BY created_at ASC;

-- ---------------------------------------------------------------------------
-- drafts awaiting clarification (never included in any aggregate)
-- ---------------------------------------------------------------------------

-- name: CreateDraft :one
INSERT INTO drafts (user_id, conversation_id, payload, missing_fields)
VALUES (
    sqlc.arg('user_id'),
    sqlc.narg('conversation_id'),
    sqlc.arg('payload'),
    sqlc.arg('missing_fields')
)
RETURNING *;

-- name: GetPendingDraft :one
SELECT *
FROM drafts
WHERE user_id = $1
  AND status = 'pending'
  AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateDraft :one
UPDATE drafts
SET payload = sqlc.arg('payload'),
    missing_fields = sqlc.arg('missing_fields')
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND status = 'pending'
RETURNING *;

-- name: ResolveDraft :exec
UPDATE drafts
SET status = 'resolved'
WHERE id = $1
  AND user_id = $2
  AND status = 'pending';
