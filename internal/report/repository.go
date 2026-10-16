package report

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/internal/db"
)

// CategoryRow is one (currency, type, category) total straight from the database.
type CategoryRow struct {
	Currency string
	Type string 
	CategoryID uuid.UUID
	CategoryName string
	Total int64
	Count int64
}

// ExpenseRow is last month's total spending in one currency.
type ExpenseRow struct {
	Currency string
	Expenses int64
}

type Repository interface{
	// monthly reports this months category totals and last month expense total
	Monthly(ctx context.Context, userID uuid.UUID, from, to, prevFrom, prevTo time.Time) ([]CategoryRow, []ExpenseRow, error)
	// Categories returns category totals for from <= date < to.
	Categories(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]CategoryRow, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRepository(pool *pgxpool.Pool, q *db.Queries) *PGRepository {
	return &PGRepository{pool: pool, q: q}
}

var _ Repository = (*PGRepository)(nil)

func (r *PGRepository) Monthly(ctx context.Context, userID uuid.UUID, from, to, prevFrom, prevTo time.Time) ([]CategoryRow, []ExpenseRow, error){
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, fmt.Errorf("begin snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := r.q.WithTx(tx)

	current, err := categoryRows(ctx, qtx, userID, from, to)
	if err != nil {
		return nil, nil, err
	}

	prevTotals, err := qtx.DashboardMonthTotals(ctx, db.DashboardMonthTotalsParams{
		UserID: userID, FromDate: prevFrom, ToDate: prevTo,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("previous month: %w", err)
	}
	previous := make([]ExpenseRow, 0, len(prevTotals))
	for _, p := range prevTotals {
		previous = append(previous, ExpenseRow{Currency: p.Currency, Expenses: p.Expenses})
	}
	return current, previous, nil
}

func (r *PGRepository) Categories(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]CategoryRow, error) {
	return categoryRows(ctx, r.q, userID, from, to)
}

func categoryRows(ctx context.Context, q *db.Queries, userID uuid.UUID, from, to time.Time) ([]CategoryRow, error) {
	rows, err := q.ReportCategoryTotals(ctx, db.ReportCategoryTotalsParams{
		UserID: userID, FromDate: from, ToDate: to,
	})
	if err != nil {
		return nil, fmt.Errorf("category totals: %w", err)
	}
	out := make([]CategoryRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, CategoryRow{
			Currency: row.Currency,
			Type: row.Type,
			CategoryID: row.CategoryID,
			CategoryName: row.CategoryName,
			Total: row.Total,
			Count: row.TxCount,
		})
	}
	return out, nil
}