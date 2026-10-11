package category

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidName = errors.New("name must be between 2 and 50 characters")
	ErrInvalidType = errors.New("type must be income or expense")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Category, error) {
	cats, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return cats, nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name, catType string) (Category, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 50 {
		return Category{}, ErrInvalidName
	}
	if !validType(catType) {
		return Category{}, ErrInvalidType
	}

	cat, err := s.repo.Create(ctx, userID, name, catType)
	if err != nil {
		if errors.Is(err, ErrNameTaken) {
			return Category{}, err
		}
		return Category{}, fmt.Errorf("create category: %w", err)
	}
	return cat, nil
}