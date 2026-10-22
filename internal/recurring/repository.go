package recurring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/internal/db"
	"spendWise/internal/transaction"
)

var ErrNotFound = errors.New("recurring transaction not found")

// NewRule carries the validated values for a new rule.
type NewRule struct {
	AccountID  uuid.UUID
	CategoryID uuid.UUID
	Type  string
	Amount int64
	Note string
	Frequency  string
	StartDate  time.Time
	EndDate *time.Time
}

type Repository interface{
	Create(ctx context.Context, userID uuid.UUID, in NewRule) (Rule, error)
	List(ctx context.Context, userID uuid.UUID) ([]Rule, error)
	// update locks the rule, and hands its current state to mutate, then saves what mutate returns all in one db transaction
	Update(ctx context.Context, userID, id uuid.UUID, mutate func(Rule) (Rule, error)) (Rule, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

type PGRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRepository(pool *pgxpool.Pool, q *db.Queries) *PGRepository {
	return &PGRepository{pool: pool, q: q}
}

var _ Repository = (*PGRepository)(nil)

func (r *PGRepository) Create(ctx context.Context, userID uuid.UUID, in NewRule) (Rule, error){
	if _, err := r.q.GetAccountForUser(ctx, db.GetAccountForUserParams{ID: in.AccountID, UserID: userID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Rule{}, transaction.ErrAccountNotFound
		}
		return Rule{}, fmt.Errorf("get account: %w", err)
	}
	cat, err := r.q.GetCategoryForUser(ctx, db.GetCategoryForUserParams{ID: in.CategoryID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Rule{}, transaction.ErrCategoryNotFound
		}
		return Rule{}, fmt.Errorf("get category: %w", err)
	}
	if cat.Type != in.Type {
		return Rule{}, transaction.ErrCategoryTypeMismatch
	}

	var end time.Time // zero value = no end date
	if in.EndDate != nil {
		end = *in.EndDate
	}
	id, err := r.q.InsertRecurring(ctx, db.InsertRecurringParams{
		UserID: userID,
		AccountID: in.AccountID,
		CategoryID: in.CategoryID,
		Type: in.Type,
		Amount: in.Amount,
		Note: in.Note,
		Frequency: in.Frequency,
		StartDate: in.StartDate,
		EndDate: end,
	})
	if err != nil {
		return Rule{}, fmt.Errorf("insert rule: %w", err)
	}

	row, err := r.q.GetRecurringForUser(ctx, db.GetRecurringForUserParams{ID: id, UserID: userID})
	if err != nil {
		return Rule{}, fmt.Errorf("load rule: %w", err)
	}
	return fromRow(db.ListRecurringForUserRow(row)), nil
}

func (r *PGRepository) List(ctx context.Context, userID uuid.UUID) ([]Rule, error){
	rows, err := r.q.ListRecurringForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Rule, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

func (r *PGRepository) Update(ctx context.Context, userID, id uuid.UUID, mutate func(Rule) (Rule, error)) (Rule, error){
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Rule{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := r.q.WithTx(tx)

	// lock first: if a worker is processing this rule atm, we wait for it to finish and then edit the up-to-date rule, never a stale copy
	row, err := qtx.LockRecurringForUser(ctx, db.LockRecurringForUserParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Rule{}, ErrNotFound
		}
		return Rule{}, fmt.Errorf("lock rule: %w", err)
	}

	updated, err := mutate(fromRow(db.ListRecurringForUserRow(row)))
	if err != nil {
		return Rule{}, err
	}

	var end time.Time
	if updated.EndDate != nil {
		end = *updated.EndDate
	}
	if _, err := qtx.UpdateRecurring(ctx, db.UpdateRecurringParams{
		ID: id,
		UserID: userID,
		Amount: updated.Amount,
		Note: updated.Note,
		EndDate: end,
		Status: updated.Status,
		NextIndex: int32(updated.NextIndex),
		NextRunOn: updated.NextRunOn,
	}); err != nil {
		return Rule{}, fmt.Errorf("update rule: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Rule{}, fmt.Errorf("commit: %w", err)
	}
	updated.LastError = "" // editing a rule clears its old error and lets the worker retry it
	return updated, nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, id uuid.UUID) error{
	n, err := r.q.DeleteRecurring(ctx, db.DeleteRecurringParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}


// fromRow converts a database row to a Rule
func fromRow(row db.ListRecurringForUserRow)Rule{
	rule := Rule{
		ID: row.ID,
		AccountID: row.AccountID,
		CategoryID: row.CategoryID,
		CategoryName: row.CategoryName,
		Type: row.Type,
		Amount: row.Amount,
		Note: row.Note,
		Frequency: row.Frequency,
		StartDate: row.StartDate,
		NextIndex: int(row.NextIndex),
		NextRunOn: row.NextRunOn,
		Status: row.Status,
		Runs: int(row.Runs),
		LastError: row.LastError,
		CreatedAt: row.CreatedAt,
	}
	if !row.EndDate.IsZero() {
		end := row.EndDate
		rule.EndDate = &end
	}
	return rule
}