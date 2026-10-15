package budget

import(
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"spendWise/internal/db"
)

var(
	ErrNotFound = errors.New("budget not found")
	ErrAlreadyExists = errors.New("a budget for this category already exists")
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryNotExpense = errors.New("budgets can only be set on expense categories")
)

const pgUniqueViolation = "23505"

// NewBudget carries the validated values for a new budget
type NewBudget struct{
	CategoryID uuid.UUID
	Amount int64
	Threshold int
}

type Repository interface{
	Create(ctx context.Context, userID uuid.UUID, in NewBudget) (uuid.UUID, error)
	List(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]Budget, error)
	Get(ctx context.Context, userID, id uuid.UUID, from, to time.Time) (Budget, error)
	Update(ctx context.Context, userID, id uuid.UUID, amount int64, threshold int) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

type PGRepository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *PGRepository { return &PGRepository{q: q} }

var _ Repository = (*PGRepository)(nil)

func (r *PGRepository) Create(ctx context.Context, userID uuid.UUID, in NewBudget) (uuid.UUID, error){
	// a category budget needs a category that the user can see and that tracks spending
	if in.CategoryID != uuid.Nil {
		cat, err := r.q.GetCategoryForUser(ctx, db.GetCategoryForUserParams{ID: in.CategoryID, UserID: userID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return uuid.Nil, ErrCategoryNotFound
			}
			return uuid.Nil, fmt.Errorf("get category: %w", err)
		}
		if cat.Type != "expense" {
			return uuid.Nil, ErrCategoryNotExpense
		}
	}

	id, err := r.q.InsertBudget(ctx, db.InsertBudgetParams{
		UserID: userID,
		CategoryID: in.CategoryID,
		Amount: in.Amount,
		AlertThreshold: int32(in.Threshold),
	})
	if err != nil{
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return uuid.Nil, ErrAlreadyExists
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (r *PGRepository) List(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]Budget, error){
	rows, err := r.q.ListBudgetsWithSpending(ctx, db.ListBudgetsWithSpendingParams{
		UserID: userID, FromDate: from, ToDate: to,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Budget, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row.ID, row.CategoryID, row.CategoryName, row.Amount, row.AlertThreshold, row.CreatedAt, row.Spent))
	}
	return out, nil
}

func (r *PGRepository) Get(ctx context.Context, userID, id uuid.UUID, from, to time.Time) (Budget, error){
	row, err := r.q.GetBudgetWithSpending(ctx, db.GetBudgetWithSpendingParams{
		ID: id, UserID: userID, FromDate: from, ToDate: to,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Budget{}, ErrNotFound
		}
		return Budget{}, err
	}
	return fromRow(row.ID, row.CategoryID, row.CategoryName, row.Amount, row.AlertThreshold, row.CreatedAt, row.Spent), nil
}

func (r *PGRepository) Update(ctx context.Context, userID, id uuid.UUID, amount int64, threshold int) error {
	n, err := r.q.UpdateBudget(ctx, db.UpdateBudgetParams{
		ID: id, UserID: userID, Amount: amount, AlertThreshold: int32(threshold),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := r.q.DeleteBudget(ctx, db.DeleteBudgetParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// fromRow converts Query columns into a budget.
func fromRow(id, categoryID uuid.UUID, categoryName string, amount int64, threshold int32, createdAt time.Time, spent int64) Budget{
	b := Budget{
		ID: id,
		CategoryName: categoryName,
		Amount: amount,
		AlertThreshold: int(threshold),
		CreatedAt: createdAt,
		Spent: spent,
	}
	if categoryID != uuid.Nil{
		b.CategoryID = &categoryID
	} else{
		b.CategoryName = "All spending"
	}
	return b
}