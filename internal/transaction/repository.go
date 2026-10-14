package transaction

import(
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/internal/db"
)

var(
	ErrAccountNotFound = errors.New("account not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryTypeMismatch = errors.New("category type does not match the transaction type")
)

//new transaction carries the validated values needed to record a transaction
type NewTransaction struct{
	AccountID uuid.UUID
	CategoryID uuid.UUID
	Type string
	Amount int64
	Note string
	Date time.Time
}

// listFilter narrrows a history query
type ListFilter struct{
	AccountID uuid.UUID 
	CategoryID uuid.UUID
	Type string
	From, To time.Time
	Limit int
	Offset int
}

type Repository interface{
	Create(ctx context.Context, userID uuid.UUID, in NewTransaction) (Transaction, error)
	List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Transaction, error)
}


type PGRepository struct{
	pool *pgxpool.Pool
	q *db.Queries
}

func NewRepository(pool *pgxpool.Pool, q *db.Queries) *PGRepository {
	return &PGRepository{pool: pool, q: q}
}

var _ Repository = (*PGRepository)(nil)

// create records a transaction and updates the account balance as one unit
func (r *PGRepository) Create(ctx context.Context, userID uuid.UUID, in NewTransaction) (Transaction, error){
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, fmt.Errorf("begin tx: %w", err)
	}
	// returning before comit undoes everything. after a successful comit, it does nothing, so its best to defer
	defer func(){ _ = tx.Rollback(ctx) }()
	qtx := r.q.WithTx(tx)  // the same queries, now running inside this transaction

	// 1. lock the account row(prove it belongs to this user). concurrent requests for the same account will queue here
	if _, err := qtx.LockAccountForUser(ctx, db.LockAccountForUserParams{ID: in.AccountID, UserID: userID}); err != nil{
		if errors.Is(err, pgx.ErrNoRows){
			return Transaction{}, ErrAccountNotFound
		}
		return Transaction{}, fmt.Errorf("lock account: %w", err)
	}

	// 2. the category must be built-in one or one of this user's, and its type must match the transaction
	cat, err := qtx.GetCategoryForUser(ctx, db.GetCategoryForUserParams{ID: in.CategoryID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, ErrCategoryNotFound
		}
		return Transaction{}, fmt.Errorf("get category: %w", err)
	}
	if cat.Type != in.Type {
		return Transaction{}, ErrCategoryTypeMismatch
	}

	// 3. store the transaction

	row, err := qtx.InsertTransaction(ctx, db.InsertTransactionParams{
		UserID: userID,
		AccountID: in.AccountID,
		CategoryID: in.CategoryID,
		Type: in.Type,
		Amount: in.Amount,
		Note: in.Note,
		OccurredOn: in.Date,
	})
	if err != nil{
		return Transaction{}, fmt.Errorf("inserct transaction: %w", err)
	}

	// 4. move the balance: income adds, expense substracts
	delta := in.Amount
	if in.Type == TypeExpense{
		delta = -delta
	}
	if err := qtx.AdjustAccountBalance(ctx, db.AdjustAccountBalanceParams{
		Delta: delta, ID: in.AccountID, UserID: userID,
	}); err != nil {
		return Transaction{}, fmt.Errorf("adjust balance: %w", err)
	}

	// 5. makes it permanent
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, fmt.Errorf("commit: %w", err)
	}

	return Transaction{
		ID: row.ID,
		AccountID: row.AccountID,
		CategoryID: row.CategoryID,
		Type: row.Type,
		Amount: row.Amount,
		Note: row.Note,
		Date: row.OccurredOn.Format(dateLayout),
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *PGRepository) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Transaction, error){
	rows, err := r.q.ListTransactionsForUser(ctx, db.ListTransactionsForUserParams{
		UserID: userID,
		AccountID: f.AccountID,
		CategoryID: f.CategoryID,
		Type: f.Type,
		FromDate: f.From,
		ToDate: f.To,
		PageLimit: int32(f.Limit),
		PageOffset: int32(f.Offset),
	})
	if err != nil{
		return nil, err
	}

	out := make([]Transaction, 0, len(rows))
	for _, row := range rows{
		out = append(out, Transaction{
			ID: row.ID,
			AccountID: row.AccountID,
			CategoryID: row.CategoryID,
			CategoryName: row.CategoryName,
			Type: row.Type,
			Amount: row.Amount,
			Note: row.Note,
			Date: row.OccurredOn.Format(dateLayout),
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}