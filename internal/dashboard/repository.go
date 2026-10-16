package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/internal/budget"
	"spendWise/internal/db"
	"spendWise/internal/transaction"
)

// BalanceRow and TotalsRow are the raw per-currency numbers.
type BalanceRow struct {
	Currency string
	Balance  int64
}

type TotalsRow struct {
	Currency string
	Income   int64
	Expenses int64
}

// snapshot is evrything the dashboard shows
type Snapshot struct {
	Balances []BalanceRow
	Totals []TotalsRow
	Recent []transaction.Transaction
	Budgets []budget.Budget
	UnreadAlerts int
}

type Repository interface {
	Load(ctx context.Context, userID uuid.UUID, from, to time.Time, recentLimit int) (Snapshot, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRepository(pool *pgxpool.Pool, q *db.Queries) *PGRepository {
	return &PGRepository{pool: pool, q: q}
}

var _ Repository = (*PGRepository)(nil)

func (r *PGRepository) Load(ctx context.Context, userID uuid.UUID, from, to time.Time, recentLimit int) (Snapshot, error){
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("begin snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // nothing to commit; this just ends the read

	qtx := r.q.WithTx(tx)

	var s Snapshot

	balances, err := qtx.DashboardBalances(ctx, userID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("balances: %w", err)
	}
	for _, b := range balances {
		s.Balances = append(s.Balances, BalanceRow{Currency: b.Currency, Balance: b.Balance})
	}

	totals, err := qtx.DashboardMonthTotals(ctx, db.DashboardMonthTotalsParams{
		UserID: userID, FromDate: from, ToDate: to,
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("month totals: %w", err)
	}
	for _, t := range totals {
		s.Totals = append(s.Totals, TotalsRow{Currency: t.Currency, Income: t.Income, Expenses: t.Expenses})
	}

	// most recent transactions overall
	recent, err := qtx.ListTransactionsForUser(ctx, db.ListTransactionsForUserParams{
		UserID:  userID,
		AccountID: uuid.Nil, // no account filter
		CategoryID: uuid.Nil, // no category filter
		Type:  "",       // income and expense
		FromDate: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		ToDate: time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC),
		PageLimit:  int32(recentLimit),
		PageOffset: 0,
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("recent transactions: %w", err)
	}
	s.Recent = make([]transaction.Transaction, 0, len(recent))
	for _, row := range recent {
		s.Recent = append(s.Recent, transaction.Transaction{
			ID:  row.ID,
			AccountID: row.AccountID,
			CategoryID: row.CategoryID,
			CategoryName: row.CategoryName,
			Type:  row.Type,
			Amount: row.Amount,
			Note: row.Note,
			Date: row.OccurredOn.Format("2006-01-02"),
			CreatedAt: row.CreatedAt,
		})
	}

	// Reuse the budget repository, bound to this same snapshot transaction.
	s.Budgets, err = budget.NewRepository(qtx).List(ctx, userID, from, to)
	if err != nil {
		return Snapshot{}, fmt.Errorf("budgets: %w", err)
	}

	unread, err := qtx.CountUnreadAlerts(ctx, userID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("unread alerts: %w", err)
	}
	s.UnreadAlerts = int(unread)

	return s, nil
}