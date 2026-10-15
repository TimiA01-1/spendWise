package alert

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const (
	defaultLimit = 20
	maxLimit = 100
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]Alert, error){
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	alerts, err := s.repo.List(ctx, userID, unreadOnly, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	return alerts, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("mark alert read: %w", err)
	}
	return nil
}