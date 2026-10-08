package user

import(
	"context"

	"github.com/google/uuid"
)

type Service struct{
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) GetProfile(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.GetByID(ctx, id)
}