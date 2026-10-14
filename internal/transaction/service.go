package transaction

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	defaultLimit = 20
	maxLimit = 100
	maxNoteLen = 500
)

var (
	ErrInvalidType = errors.New("type must be income or expense")
	ErrInvalidAmount = errors.New("amount must be a positive whole number of kobo, at most 1000000000000")
	ErrNoteTooLong = errors.New("note must be at most 500 characters")
	ErrInvalidRange = errors.New("'from' date must not be after 'to' date")
)

type Service struct{
	repo Repository
	now  func() time.Time // injectable for tests
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

type CreateInput struct{
	AccountID uuid.UUID
	CategoryID uuid.UUID
	Type string
	Amount int64
	Note string
	Date *time.Time
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (Transaction, error){
	if !validType(in.Type){
		return Transaction{}, ErrInvalidType
	}
	if in.Amount <= 0 || in.Amount > MaxAmount {
		return Transaction{}, ErrInvalidAmount
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > maxNoteLen {
		return Transaction{}, ErrNoteTooLong
	}

	date := s.today()
	if in.Date != nil{
		date = in.Date.UTC().Truncate(24 * time.Hour)
	}
	t, err := s.repo.Create(ctx, userID, NewTransaction{
		AccountID: in.AccountID,
		CategoryID: in.CategoryID,
		Type: in.Type,
		Amount: in.Amount,
		Note: note,
		Date: date,
	})
	if err != nil{
		//if isDomainError(err){
		//	return Transaction{}, err
		//}
		return Transaction{}, passThrough("create transaction", err)
	}
	return t, nil
}

// updateimpt uses pointers to tell "field not sent" from "sent"
type UpdateInput struct{
	AccountID *uuid.UUID
	CategoryID *uuid.UUID
	Type *string
	Amount *int64
	Note *string
	Date *time.Time
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Transaction, error) {
	t, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return Transaction{}, passThrough("get transaction", err)
	}
	return t, nil
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (Transaction, error){
	if in.Type != nil && !validType(*in.Type) {
		return Transaction{}, ErrInvalidType
	}
	if in.Amount != nil && (*in.Amount <= 0 || *in.Amount > MaxAmount) {
		return Transaction{}, ErrInvalidAmount
	}
	up := UpdateTransaction{
		AccountID: in.AccountID,
		CategoryID: in.CategoryID,
		Type: in.Type,
		Amount: in.Amount,
	}
	if in.Note != nil {
		note := strings.TrimSpace(*in.Note)
		if utf8.RuneCountInString(note) > maxNoteLen {
			return Transaction{}, ErrNoteTooLong
		}
		up.Note = &note
	}
	if in.Date != nil {
		d := in.Date.UTC().Truncate(24 * time.Hour)
		up.Date = &d
	}

	t, err := s.repo.Update(ctx, userID, id, up)
	if err != nil {
		return Transaction{}, passThrough("update transaction", err)
	}
	return t, nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return passThrough("delete transaction", err)
	}
	return nil
}

type ListInput struct{
	AccountID uuid.UUID
	CategoryID uuid.UUID
	Type string
	From, To *time.Time
	Limit int
	Offset int
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, in ListInput) ([]Transaction, error){
	if in.Type != "" && !validType(in.Type) {
		return nil, ErrInvalidType
	}
	from := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	if in.From != nil {
		from = *in.From
	}
	if in.To != nil {
		to = *in.To
	}
	if from.After(to) {
		return nil, ErrInvalidRange
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	txs, err := s.repo.List(ctx, userID, ListFilter{
		AccountID: in.AccountID,
		CategoryID: in.CategoryID,
		Type: in.Type,
		From: from,
		To: to,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil{
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return txs, nil
}

func (s *Service) today() time.Time{
	return s.now().UTC().Truncate(24 * time.Hour)
}

// passThrough keeps domain errors unchanged
func passThrough(op string, err error) error {
	if errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrAccountNotFound) ||
		errors.Is(err, ErrCategoryNotFound) ||
		errors.Is(err, ErrCategoryTypeMismatch) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}

//func isDomainError(err error) bool{
//	return errors.Is(err, ErrAccountNotFound) ||
//		errors.Is(err, ErrCategoryNotFound) ||
//		errors.Is(err, ErrCategoryTypeMismatch)
//}