package account

import(
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const defaultCurrency = "NGN"

var(
	ErrInvalidName = errors.New("name must be between 2 and 100 characters")
	ErrInvalidType = errors.New("type must be bank, cash or wallet")
	ErrNegativeBalance = errors.New("opening balance cannot be negative")
)

type Service struct{
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

type CreateInput struct{
	Name string
	Type string
	Currency string
	OpeningBalance int64
}

//UpdateInuput uses pointers to tell "field not sent" from "field sent"
type UpdateInput struct{
	Name *string
	Type *string
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput)(Account, error){
	name := strings.TrimSpace(in.Name)
	if err := validateName(name); err != nil{
		return Account{}, err
	}
	//if n := utf8.RuneCountInString(name); n < 2 || n > 100{
	//	return Account{}, ErrInvalidType
	//}
	if !validType(in.Type){
		return Account{}, ErrInvalidType
	}
	if in.OpeningBalance < 0{
		return Account{}, ErrNegativeBalance
	}

	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == ""{
		currency = defaultCurrency
	}
	acc, err := s.repo.Create(ctx, userID, NewAccount{
		Name: name,
		Type: in.Type,
		Currency: currency,
		Balance: in.OpeningBalance,
	})
	if err != nil{
		return Account{}, wrap("create account", err)
		//if errors.Is(err, ErrNameTaken){
		//	return Account{}, err
		//}
		//return Account{}, fmt.Errorf("Create account: %w", err)
	}
	return acc, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID)([]Account, error){
	accounts, err := s.repo.ListByUser(ctx, userID)
	if err != nil{
		return nil, wrap("list accounts", err)
		//return nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Account, error) {
	acc, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return Account{}, wrap("get account", err)
	}
	return acc, nil
}

// update only applies to fields that were sent and keeps the rest unchanged
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (Account, error){
	current, err := s.repo.GetByID(ctx, userID, id)
	if err != nil{
		return Account{}, wrap("get account", err)
	}
	name, accType := current.Name, current.Type
	if in.Name != nil{
		name = strings.TrimSpace(*in.Name)
		if err := validateName(name); err != nil {
			return Account{}, err
		}
	}
	if in.Type != nil{
		if !validType(*in.Type) {
			return Account{}, ErrInvalidType
		}
		accType = *in.Type
	}
	acc, err := s.repo.Update(ctx, userID, id, name, accType)
	if err != nil{
		return Account{}, wrap("update account", err)
	}
	return acc, nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return wrap("delete account", err)
	}
	return nil
}

func validateName(name string) error {
	if n := utf8.RuneCountInString(name); n < 2 || n > 100 {
		return ErrInvalidName
	}
	return nil
}

// wrap adds context to unexpected errors
func wrap(op string, err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrNameTaken) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}