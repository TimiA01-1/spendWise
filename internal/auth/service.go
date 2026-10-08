package auth

import(
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"spendWise/internal/user"
)

const maxPasswordBytes = 72

var (
	ErrEmailTaken = errors.New("email already registered")
	ErrPasswordTooLong = errors.New("password must be at most 72 bytes")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type UserStore interface{
	Create(ctx context.Context, name, email, passwordHash string)(user.User, error)
	GetByEmail(ctx context.Context, email string)(user.User, error)
}

type Service struct{
	users UserStore
	tokens *TokenManager
	dummyHash []byte  // compared against when email is unknown
}

func NewService(users UserStore, tokens *TokenManager) *Service{
	dummy, _ := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	return &Service{users: users, tokens: tokens, dummyHash: dummy}
}

type RegisterInput struct{
	Name string
	Email string
	Password string
}

type LoginResult struct{
	User user.User `json:"user"`
	Token string `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) Register(ctx context.Context, in RegisterInput)(user.User,error){
	if len(in.Password) > maxPasswordBytes{
		return user.User{}, ErrPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil{
		return user.User{}, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.users.Create(
		ctx,
		strings.TrimSpace(in.Name),
		normalizeEmail(in.Email),
		string(hash),
	)
	if err != nil{
		if errors.Is(err, user.ErrEmailTaken){
			return user.User{}, ErrEmailTaken
		}
		return user.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	u, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// Do the same amount of work as a real check, so response time
			// doesn't reveal whether an email is registered.
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, exp, err := s.tokens.Generate(u.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Token: token, ExpiresAt: exp}, nil
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}