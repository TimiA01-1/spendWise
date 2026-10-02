package user

import(
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"spendWise/internal/db"
)

var(
	ErrNotFound = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")
)

const(
	pgUniqueViolation = "23505"
	emailUniqueIndex  = "users_email_key"
)

// repository is storage contract for users
type Repository interface{
	Create(ctx context.Context, name, email, passwordHash string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
}

// pgrepository implements repository using the sqlc generated queries
type PGRepository struct{
	q *db.Queries
}

func NewRepository(q *db.Queries) *PGRepository{ return &PGRepository{q: q} }

// compile-time check that pgrepository satisfies repository
var _ Repository = (*PGRepository)(nil)

func (r *PGRepository) Create(ctx context.Context, name, email, passwordHash string)(User, error){
	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Name: name,
		Email: email,
		PasswordHash: passwordHash,
	})
	if err != nil{
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == emailUniqueIndex {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return fromRow(row), nil
}

func (r *PGRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, mapErr(err)
	}
	return fromRow(row), nil
}

func (r *PGRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return User{}, mapErr(err)
	}
	return fromRow(row), nil
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func fromRow(r db.User) User{
	return User{
		ID: r.ID,
		Name: r.Name,
		Email: r.Email,
		Currency: r.Currency,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		PasswordHash: r.PasswordHash,
	}
}