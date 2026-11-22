package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// UpdateMeInput is the PATCH /me body.
type UpdateMeInput struct {
	DisplayName        *string `json:"displayName"`
	Timezone           *string `json:"timezone"`
	Locale             *string `json:"locale"`
	CompleteOnboarding bool    `json:"completeOnboarding"`
}

// GetMe returns the profile row.
func (s *Service) GetMe(ctx context.Context, q *generated.Queries, userID uuid.UUID) (UserDTO, error) {
	user, err := q.GetUserByID(ctx, pgUUIDValue(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return UserDTO{}, NotFound("User")
	}
	if err != nil {
		return UserDTO{}, mapDBError(err)
	}
	return userDTO(user), nil
}

// UpdateMe changes profile preferences.
func (s *Service) UpdateMe(ctx context.Context, q *generated.Queries, userID uuid.UUID, in UpdateMeInput) (UserDTO, error) {
	displayName := optionalString(in.DisplayName)
	if displayName == nil && in.DisplayName != nil {
		return UserDTO{}, Invalid("displayName", "Nama tampilan wajib diisi")
	}
	locale := optionalString(in.Locale)
	if locale != nil && *locale != "id-ID" && *locale != "en-US" {
		return UserDTO{}, Invalid("locale", "Locale harus id-ID atau en-US")
	}
	timezone := optionalString(in.Timezone)
	if timezone != nil {
		if _, err := time.LoadLocation(*timezone); err != nil {
			return UserDTO{}, Invalid("timezone", "Timezone tidak valid")
		}
	}

	user, err := q.UpdateUserPreferences(ctx, generated.UpdateUserPreferencesParams{
		DisplayName: displayName,
		Timezone:    timezone,
		Locale:      locale,
		ID:          pgUUIDValue(userID),
	})
	if err != nil {
		return UserDTO{}, mapDBError(err)
	}
	if in.CompleteOnboarding {
		if err := q.MarkUserOnboarded(ctx, pgUUIDValue(userID)); err != nil {
			return UserDTO{}, mapDBError(err)
		}
		user.OnboardedAt.Valid = true
		user.OnboardedAt.Time = s.NowUTC()
	}
	return userDTO(user), nil
}

// UserTimezone resolves the user's timezone, defaulting to Asia/Jakarta.
func (s *Service) UserTimezone(ctx context.Context, q *generated.Queries, userID uuid.UUID) (string, error) {
	user, err := q.GetUserByID(ctx, pgUUIDValue(userID))
	if err != nil {
		return "", mapDBError(err)
	}
	tz := strings.TrimSpace(user.Timezone)
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	return tz, nil
}
