package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// CreateCategoryInput is the POST /categories body.
type CreateCategoryInput struct {
	Name  string  `json:"name"`
	Kind  string  `json:"kind"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}

// UpdateCategoryInput is the PATCH /categories/{id} body.
type UpdateCategoryInput struct {
	Name  *string `json:"name"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}

// ListCategories returns active categories, optionally filtered by kind.
func (s *Service) ListCategories(ctx context.Context, q *generated.Queries, userID uuid.UUID, kind string) ([]CategoryDTO, error) {
	var kindParam *string
	if k := strings.TrimSpace(kind); k != "" {
		if k != "income" && k != "expense" {
			return nil, Invalid("kind", "Kind harus income atau expense")
		}
		kindParam = &k
	}
	rows, err := q.ListCategoriesByUser(ctx, generated.ListCategoriesByUserParams{
		UserID: pgUUIDValue(userID),
		Kind:   kindParam,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]CategoryDTO, 0, len(rows))
	for _, c := range rows {
		out = append(out, categoryDTO(c))
	}
	return out, nil
}

// CreateCategory adds a user category.
func (s *Service) CreateCategory(ctx context.Context, q *generated.Queries, userID uuid.UUID, in CreateCategoryInput) (CategoryDTO, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return CategoryDTO{}, Invalid("name", "Nama kategori wajib diisi")
	}
	if in.Kind != "income" && in.Kind != "expense" {
		return CategoryDTO{}, Invalid("kind", "Kind harus income atau expense")
	}
	icon := optionalString(in.Icon)
	color := optionalString(in.Color)
	if color != nil && !strings.HasPrefix(*color, "#") {
		return CategoryDTO{}, Invalid("color", "Warna harus format hex, mis. #10B981")
	}

	category, err := q.CreateCategory(ctx, generated.CreateCategoryParams{
		UserID: pgUUIDValue(userID),
		Name:   name,
		Kind:   in.Kind,
		Icon:   icon,
		Color:  color,
	})
	if err != nil {
		return CategoryDTO{}, mapDBError(err)
	}
	return categoryDTO(category), nil
}

// UpdateCategory renames or restyles a category.
func (s *Service) UpdateCategory(ctx context.Context, q *generated.Queries, userID uuid.UUID, categoryID string, in UpdateCategoryInput) (CategoryDTO, error) {
	id, err := parseUUID(categoryID)
	if err != nil {
		return CategoryDTO{}, err
	}
	name := optionalString(in.Name)
	if name == nil && in.Name != nil {
		return CategoryDTO{}, Invalid("name", "Nama kategori wajib diisi")
	}
	icon := optionalString(in.Icon)
	color := optionalString(in.Color)
	if color != nil && !strings.HasPrefix(*color, "#") {
		return CategoryDTO{}, Invalid("color", "Warna harus format hex, mis. #10B981")
	}

	category, err := q.UpdateCategory(ctx, generated.UpdateCategoryParams{
		Name:   name,
		Icon:   icon,
		Color:  color,
		ID:     pgUUIDValue(id),
		UserID: pgUUIDValue(userID),
	})
	if err != nil {
		return CategoryDTO{}, mapDBError(err)
	}
	return categoryDTO(category), nil
}

// ArchiveCategory soft-deletes a category. Default categories may be archived
// but never deleted (plan.md §5.3).
func (s *Service) ArchiveCategory(ctx context.Context, q *generated.Queries, userID uuid.UUID, categoryID string) error {
	id, err := parseUUID(categoryID)
	if err != nil {
		return err
	}
	category, err := q.GetCategoryForUser(ctx, generated.GetCategoryForUserParams{
		ID:     pgUUIDValue(id),
		UserID: pgUUIDValue(userID),
	})
	if err != nil || category.ArchivedAt.Valid {
		return NotFound("Kategori")
	}
	return mapDBError(q.ArchiveCategory(ctx, generated.ArchiveCategoryParams{
		ID:     pgUUIDValue(id),
		UserID: pgUUIDValue(userID),
	}))
}
