package category

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"spendWise/internal/db"
)

var ErrNameTaken = errors.New("a category with this name and type already exists")

const (
	pgUniqueViolation = "23505"
	userNameIndex = "categories_user_name_key"
)

type Repository interface{
	ListForUser(ctx context.Context, userID uuid.UUID) ([]Category, error)
	Create(ctx context.Context, userID uuid.UUID, name, catType string) (Category, error)
}

type PGRepository struct{
	q *db.Queries
}

func NewRepository(q *db.Queries) *PGRepository { return &PGRepository{q: q} }

var _ Repository = (*PGRepository)(nil)

// ListForUser returns the built-in categories and this users own
func (r *PGRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]Category, error) {
	rows, err := r.q.ListCategoriesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, Category{
			ID: row.ID, Name: row.Name, Type: row.Type,
			IsDefault: row.IsDefault, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (r *PGRepository) Create(ctx context.Context, userID uuid.UUID, name, catType string) (Category, error) {
	row, err := r.q.CreateCategory(ctx, db.CreateCategoryParams{
		UserID: userID,
		Name:   name,
		Type:   catType,
	})
	if err != nil {
		// The INSERT inserts nothing when a built-in category already has this
		// name and type, which surfaces as "no rows".
		if errors.Is(err, pgx.ErrNoRows) {
			return Category{}, ErrNameTaken
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == userNameIndex {
			return Category{}, ErrNameTaken
		}
		return Category{}, err
	}
	return Category{
		ID: row.ID, Name: row.Name, Type: row.Type,
		IsDefault: row.IsDefault, CreatedAt: row.CreatedAt,
	}, nil
}