package account

import(
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"spendWise/internal/db"
)

var(
	ErrNotFound = errors.New("account not found")
	ErrNameTaken = errors.New("an account with this name already exists")
)
const(
	pgUniqueViolation = "23505"
	nameUniqueIndex = "accounts_user_name_key"
)

// new account carries the values needed to insert an account
type NewAccount struct{
	Name string
	Type string
	Currency string
	Balance int64
}

// repository is the storage contract for accounts
type Repository interface{
	Create(ctx context.Context, userID uuid.UUID, in NewAccount)(Account, error)
	ListByUser(ctx context.Context, userID uuid.UUID)([]Account, error)
	GetByID(ctx context.Context, userID, id uuid.UUID) (Account, error)
	Update(ctx context.Context, userID, id uuid.UUID, name, accType string) (Account, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

type PGRepository struct{
	q *db.Queries
}

func NewRepository(q *db.Queries) *PGRepository { return &PGRepository{q: q} }

var _ Repository = (*PGRepository)(nil)

func (r *PGRepository) Create(ctx context.Context, userID uuid.UUID, in NewAccount)(Account, error){
	row, err := r.q.CreateAccount(ctx, db.CreateAccountParams{
		UserID: userID,
		Name: in.Name,
		Type: in.Type,
		Currency: in.Currency,
		Balance: in.Balance,
	})
	if err != nil{
		if isNameTaken(err){
			return Account{}, ErrNameTaken
		}
		//var pgErr *pgconn.PgError
		//if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == nameUniqueIndex{
		//	return Account{}, ErrNameTaken
		//}
		return Account{}, err
	}
	return fromRow(row), nil
}

func (r *PGRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Account, error){
	rows, err := r.q.ListAccountsByUser(ctx, userID)
	if err != nil{
		return nil, err
	}
	out := make([]Account, 0, len(rows)) // empty slice, not nil, so JSON shows [] not null
	for _, row := range rows{
		out = append(out, fromRow(row))
	}
	return out, nil
}


// GetByID returns ErrNotFound when the account doesnt exist or when it belongs to someone else
func (r *PGRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (Account, error) {
	row, err := r.q.GetAccountForUser(ctx, db.GetAccountForUserParams{ID: id, UserID: userID})
	if err != nil {
		return Account{}, mapErr(err)
	}
	return fromRow(row), nil
}

func (r *PGRepository) Update(ctx context.Context, userID, id uuid.UUID, name, accType string) (Account, error){
	row, err := r.q.UpdateAccount(ctx, db.UpdateAccountParams{
		ID: id,
		UserID: userID,
		Name: name,
		Type: accType,
	})
	if err != nil{
		if isNameTaken(err){
			return Account{}, ErrNameTaken
		}
		return Account{}, mapErr(err)
	}
	return fromRow(row), nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := r.q.DeleteAccount(ctx, db.DeleteAccountParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isNameTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == nameUniqueIndex
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func fromRow(r db.Account) Account{
	return Account{
		ID: r.ID,
		UserID: r.UserID,
		Name: r.Name,
		Type: r.Type,
		Currency: r.Currency,
		Balance: r.Balance,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}