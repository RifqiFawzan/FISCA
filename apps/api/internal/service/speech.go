package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fisca-app/fisca/apps/api/internal/ai"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// IdempotencyReplay is returned when the same key was already completed.
type IdempotencyReplay struct {
	StatusCode int
	Body       []byte
}

// BeginIdempotency reserves the key. When done is true the caller must write
// the stored response and skip the work. A key still in progress is a conflict,
// which is what stops two concurrent duplicate writes (plan.md D-07).
func (s *Service) BeginIdempotency(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	key, endpoint string,
	payload any,
) (replay IdempotencyReplay, done bool, err error) {
	hash, err := hashPayload(payload)
	if err != nil {
		return IdempotencyReplay{}, false, err
	}
	rows, err := q.ReserveIdempotencyKey(ctx, generated.ReserveIdempotencyKeyParams{
		UserID:      pgUUIDValue(userID),
		Key:         key,
		Endpoint:    endpoint,
		RequestHash: hash,
	})
	if err != nil {
		return IdempotencyReplay{}, false, mapDBError(err)
	}
	if rows > 0 {
		return IdempotencyReplay{}, false, nil
	}

	existing, err := q.GetIdempotencyKey(ctx, generated.GetIdempotencyKeyParams{
		UserID:   pgUUIDValue(userID),
		Key:      key,
		Endpoint: endpoint,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return IdempotencyReplay{}, false, nil
	}
	if err != nil {
		return IdempotencyReplay{}, false, mapDBError(err)
	}
	if existing.RequestHash != hash {
		return IdempotencyReplay{}, false, IdempotencyMismatch()
	}
	if existing.StatusCode == 0 {
		return IdempotencyReplay{}, false, &Error{
			Status:  http.StatusConflict,
			Code:    CodeConflict,
			Message: "Permintaan dengan key ini sedang diproses",
		}
	}
	return IdempotencyReplay{StatusCode: int(existing.StatusCode), Body: existing.ResponseBody}, true, nil
}

// CompleteIdempotency stores the final response for later replays.
func (s *Service) CompleteIdempotency(ctx context.Context, q *generated.Queries, userID uuid.UUID, key, endpoint string, status int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return mapDBError(q.CompleteIdempotencyKey(ctx, generated.CompleteIdempotencyKeyParams{
		StatusCode:   int32(status),
		ResponseBody: encoded,
		UserID:       pgUUIDValue(userID),
		Key:          key,
		Endpoint:     endpoint,
	}))
}

// AbortIdempotency releases a reservation whose work failed, so the client may
// retry with the same key.
func (s *Service) AbortIdempotency(ctx context.Context, q *generated.Queries, userID uuid.UUID, key, endpoint string) {
	_ = q.DeleteIdempotencyKey(ctx, generated.DeleteIdempotencyKeyParams{
		UserID:   pgUUIDValue(userID),
		Key:      key,
		Endpoint: endpoint,
	})
}

func hashPayload(payload any) (string, error) {
	var raw []byte
	switch v := payload.(type) {
	case []byte:
		raw = v
	case json.RawMessage:
		raw = v
	case nil:
		raw = []byte("{}")
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		raw = encoded
	}
	// Re-marshal maps so member order cannot change the hash.
	var canonical any
	if err := json.Unmarshal(raw, &canonical); err == nil {
		if encoded, err := json.Marshal(canonical); err == nil {
			raw = encoded
		}
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// TranscribeAudio proxies to the configured speech-to-text provider.
func (s *Service) TranscribeAudio(ctx context.Context, filename string, audio io.Reader) (string, error) {
	if s.AI == nil || !s.AI.STTAvailable() {
		return "", ai.ErrUnavailable
	}
	return s.AI.Transcribe(ctx, filename, audio)
}
