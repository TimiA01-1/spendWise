package budget

import(
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidAmount = errors.New("amount must be a positive whole number of kobo, at most 1000000000000")
	ErrInvalidThreshold = errors.New("alert_threshold must be between 1 and 100")
	ErrInvalidMonth = errors.New("month must look like 2026-10")
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

type CreateInput struct {
	CategoryID uuid.UUID 
	Amount int64
	Threshold  *int 
}

// create stores a budget and returns it with the current months spending
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput)(Budget, error){
	if in.Amount <= 0 || in.Amount > MaxAmount {
		return Budget{}, ErrInvalidAmount
	}
	threshold := DefaultThreshold
	if in.Threshold != nil {
		threshold = *in.Threshold
	}
	if threshold < 1 || threshold > 100 {
		return Budget{}, ErrInvalidThreshold
	}

	id, err := s.repo.Create(ctx, userID, NewBudget{
		CategoryID: in.CategoryID,
		Amount: in.Amount,
		Threshold: threshold,
	})
	if err != nil {
		return Budget{}, passThrough("create budget", err)
	}
	return s.Get(ctx, userID, id, "")
}

// list returns every budget with its spending for the month
func (s *Service) List(ctx context.Context, userID uuid.UUID, month string) ([]Budget, error){
	from, to, label, err := monthRange(month, s.now())
	if err != nil {
		return nil, err
	}
	budgets, err := s.repo.List(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	for i := range budgets {
		budgets[i].Month = label
		budgets[i].Evaluate()
	}
	return budgets, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID, month string)(Budget, error){
	from, to, label, err := monthRange(month, s.now())
	if err != nil{
		return Budget{}, err
	}
	b, err := s.repo.Get(ctx, userID, id, from, to)
	if err != nil{
		return Budget{}, passThrough("get budget", err)
	}
	b.Month = label
	b.Evaluate()
	return b, nil
}

// updateInput uses pointers so we can tell "not sent" from "sent"
type UpdateInput struct{
	Amount *int64
	Threshold *int
}

func(s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput)(Budget, error){
	if in.Amount != nil && (*in.Amount <= 0 || *in.Amount > MaxAmount) {
		return Budget{}, ErrInvalidAmount
	}
	if in.Threshold != nil && (*in.Threshold < 1 || *in.Threshold > 100) {
		return Budget{}, ErrInvalidThreshold
	}

	current, err := s.Get(ctx, userID, id, "")
	if err != nil {
		return Budget{}, err
	}
	amount, threshold := current.Amount, current.AlertThreshold
	if in.Amount != nil {
		amount = *in.Amount
	}
	if in.Threshold != nil {
		threshold = *in.Threshold
	}

	if err := s.repo.Update(ctx, userID, id, amount, threshold); err != nil {
		return Budget{}, passThrough("update budget", err)
	}
	return s.Get(ctx, userID, id, "")
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return passThrough("delete budget", err)
	}
	return nil
}

// passthrough keeps domain errors unchanged and wraps anything else
func passThrough(op string, err error) error {
	if errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrAlreadyExists) ||
		errors.Is(err, ErrCategoryNotFound) ||
		errors.Is(err, ErrCategoryNotExpense) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}