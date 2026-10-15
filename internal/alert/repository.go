package alert

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"spendWise/internal/db"
)

var ErrNotFound = errors.New("alert not found")

type Repository interface {
	List(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Alert, error)
	MarkRead(ctx context.Context, userID, id uuid.UUID) error
}

type PGRepository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *PGRepository { return &PGRepository{q: q} }

var _ Repository = (*PGRepository)(nil)

func (r *PGRepository) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Alert, error){
	rows, err := r.q.ListAlertsForUser(ctx, db.ListAlertsForUserParams{
		UserID: userID,
		UnreadOnly: unreadOnly,
		PageLimit: int32(limit),
		PageOffset: int32(offset),
	})
	if err != nil{
		return nil, err
	}
	out := make([]Alert, 0, len(rows))
	for _, row := range rows{
		out = append(out, Alert{
			ID: row.ID,
			BudgetID: row.BudgetID,
			CategoryName: row.CategoryName,
			Month: row.Month.Format("2006-01"),
			Level: row.Level,
			Amount: row.Amount,
			Spent: row.Spent,
			PercentUsed: int(row.PercentUsed),
			Message: message(row.CategoryName, row.Level, int(row.PercentUsed)),
			Read: row.IsRead,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

// MarkRead is idempotent: marking an already-read alert read succeeds again.
func (r *PGRepository) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	n, err := r.q.MarkAlertRead(ctx, db.MarkAlertReadParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}