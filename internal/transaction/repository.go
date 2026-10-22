package transaction

import(
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/internal/db"
	"spendWise/internal/budget"
)

var(
	ErrNotFound = errors.New("transaction not found")
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

// updatetransaction holds the fields a PATCH may change
type UpdateTransaction struct{
	AccountID *uuid.UUID
	CategoryID *uuid.UUID
	Type *string
	Amount *int64
	Note *string
	Date *time.Time
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
	GetByID(ctx context.Context, userID, id uuid.UUID) (Transaction, error)
	Update(ctx context.Context, userID, id uuid.UUID, in UpdateTransaction) (Transaction, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
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

	t, err := RecordInTx(ctx, r.q.WithTx(tx), userID, in)
	if err != nil {
		return Transaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, fmt.Errorf("commit: %w", err)
	}
	return t, nil
}

// RecordInTx does all the work of recording a transaction INSIDE a database
func RecordInTx(ctx context.Context, qtx *db.Queries, userID uuid.UUID, in NewTransaction)(Transaction, error){
	//qtx := r.q.WithTx(tx)  // the same queries, now running inside this transaction

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
	if err := adjust(ctx, qtx, userID, in.AccountID, signed(in.Type, in.Amount)); err != nil {
		return Transaction{}, err
	}
	//delta := in.Amount
	//if in.Type == TypeExpense{
	//	delta = -delta
	//}
	//if err := qtx.AdjustAccountBalance(ctx, db.AdjustAccountBalanceParams{
	//	Delta: delta, ID: in.AccountID, UserID: userID,
	//}); err != nil {
	//	return Transaction{}, fmt.Errorf("adjust balance: %w", err)
	//}

	// 5. for epenses, check thebudgets this counts towards and record any alerts
	if in.Type == TypeExpense{
		if err := budget.RecordAlerts(ctx, qtx, userID, in.CategoryID, in.Date); err != nil {
			return Transaction{}, fmt.Errorf("record alerts: %w", err)
		}
	}

	// 6. makes it permanent
	//if err := tx.Commit(ctx); err != nil {
	//	return Transaction{}, fmt.Errorf("commit: %w", err)
	//}

	return Transaction{
		ID: row.ID,
		AccountID: row.AccountID,
		CategoryID: row.CategoryID,
		CategoryName: cat.Name,
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

func (r *PGRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (Transaction, error) {
	row, err := r.q.GetTransactionForUser(ctx, db.GetTransactionForUserParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, ErrNotFound
		}
		return Transaction{}, err
	}
	return Transaction{
		ID: row.ID,
		AccountID: row.AccountID,
		CategoryID: row.CategoryID,
		CategoryName: row.CategoryName,
		Type: row.Type,
		Amount: row.Amount,
		Note: row.Note,
		Date: row.OccurredOn.Format(dateLayout),
		CreatedAt: row.CreatedAt,
	}, nil
}

// signed is the effect a transaction has on its account balance
func signed(txType string, amount int64) int64 {
	if txType == TypeExpense {
		return -amount
	}
	return amount
}

func (r *PGRepository) Update(ctx context.Context, userID, id uuid.UUID, in UpdateTransaction) (Transaction, error){
	tx, err := r.pool.Begin(ctx)
	if err != nil{
		return Transaction{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := r.q.WithTx(tx)

	// 1. lock the transaction row and read its Current values
	old, err := qtx.LockTransactionForUser(ctx, db.LockTransactionForUserParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, ErrNotFound
		}
		return Transaction{}, fmt.Errorf("lock transaction: %w", err)
	}

	// 2. merge: take each new value if sent, otherwise, keep the old value
	accountID, categoryID, txType, amount, note, date :=
		old.AccountID, old.CategoryID, old.Type, old.Amount, old.Note, old.OccurredOn
	if in.AccountID != nil {
		accountID = *in.AccountID
	}
	if in.CategoryID != nil {
		categoryID = *in.CategoryID
	}
	if in.Type != nil {
		txType = *in.Type
	}
	if in.Amount != nil {
		amount = *in.Amount
	}
	if in.Note != nil {
		note = *in.Note
	}
	if in.Date != nil {
		date = *in.Date
	}

	// 3. lock every account involved, always in the same order
	ids := []uuid.UUID{old.AccountID}
	if accountID != old.AccountID{
		ids = append(ids, accountID)
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 })
	for _, aid := range ids {
		if _, err := qtx.LockAccountForUser(ctx, db.LockAccountForUserParams{ID: aid, UserID: userID}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Transaction{}, ErrAccountNotFound
			}
			return Transaction{}, fmt.Errorf("lock account: %w", err)
		}
	}

	// 4. The possibly new category must be visible to this user and match possibly new type
	cat, err := qtx.GetCategoryForUser(ctx, db.GetCategoryForUserParams{ID: categoryID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, ErrCategoryNotFound
		}
		return Transaction{}, fmt.Errorf("get category: %w", err)
	}
	if cat.Type != txType {
		return Transaction{}, ErrCategoryTypeMismatch
	}

	// 5. save the new values
	row, err := qtx.UpdateTransaction(ctx, db.UpdateTransactionParams{
		ID: id,
		UserID: userID,
		AccountID: accountID,
		CategoryID: categoryID,
		Type: txType,
		Amount: amount,
		Note: note,
		OccurredOn: date,
	})
	if err != nil{
		return Transaction{}, fmt.Errorf("update transaction: %w", err)
	}

	// 6. fix the balances
	oldEffect, newEffect := signed(old.Type, old.Amount), signed(txType, amount)
	if accountID == old.AccountID {
		if net := newEffect - oldEffect; net != 0 {
			if err := adjust(ctx, qtx, userID, accountID, net); err != nil {
				return Transaction{}, err
			}
		}
	} else {
		if err := adjust(ctx, qtx, userID, old.AccountID, -oldEffect); err != nil {
			return Transaction{}, err
		}
		if err := adjust(ctx, qtx, userID, accountID, newEffect); err != nil {
			return Transaction{}, err
		}
	}

	// 7. check the budgets for edited expense category and month
	if txType == TypeExpense {
		if err := budget.RecordAlerts(ctx, qtx, userID, categoryID, date); err != nil {
			return Transaction{}, fmt.Errorf("record alerts: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil{
		return Transaction{}, fmt.Errorf("commit: %w", err)
	}

	return Transaction{
		ID: row.ID,
		AccountID: row.AccountID,
		CategoryID: row.CategoryID,
		CategoryName: cat.Name,
		Type: row.Type,
		Amount: row.Amount,
		Note: row.Note,
		Date: row.OccurredOn.Format(dateLayout),
		CreatedAt: row.CreatedAt,
	}, nil
}

// delete removes a transaction and reverses its effect on the account balance
func (r *PGRepository) Delete(ctx context.Context, userID, id uuid.UUID) error{
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := r.q.WithTx(tx)

	old, err := qtx.LockTransactionForUser(ctx, db.LockTransactionForUserParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock transaction: %w", err)
	}

	if _, err := qtx.LockAccountForUser(ctx, db.LockAccountForUserParams{ID: old.AccountID, UserID: userID}); err != nil {
		return fmt.Errorf("lock account: %w", err)
	}

	if err := qtx.DeleteTransaction(ctx, db.DeleteTransactionParams{ID: id, UserID: userID}); err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}

	if err := adjust(ctx, qtx, userID, old.AccountID, -signed(old.Type, old.Amount)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func adjust(ctx context.Context, q *db.Queries, userID, accountID uuid.UUID, delta int64) error{
	if err := q.AdjustAccountBalance(ctx, db.AdjustAccountBalanceParams{
		Delta: delta, ID: accountID, UserID: userID,
	}); err != nil {
		return fmt.Errorf("adjust balance: %w", err)
	}
	return nil
}