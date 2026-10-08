package auth

import(
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"spendWise/internal/user"
)

type fakeStore struct{
	mu sync.Mutex
	users map[string]user.User
}

func newFakeStore() *fakeStore{return &fakeStore{users: map[string]user.User{}}}

func (f *fakeStore) Create(_ context.Context, name, email, hash string)(user.User, error){
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.users[email]; exists{
		return user.User{}, user.ErrEmailTaken
	}
	u := user.User{ID: uuid.New(), Name: name, Email: email, Currency: "NGN", PasswordHash: hash}
	f.users[email] = u
	return u, nil
}

func (f *fakeStore) GetByEmail(_ context.Context, email string)(user.User, error){
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[email]
	if !ok{
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

func newTestService()(*Service, *fakeStore){
	store := newFakeStore()
	return NewService(store, NewTokenManager(testSecret, time.Hour)), store
}

func TestRegister_Success(t *testing.T){
	svc, store := newTestService()

	u, err := svc.Register(context.Background(), RegisterInput{
		Name: "  King Adesanya ", Email: "  King@Example.COM ", Password: "correct-horse",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if u.Email != "king@example.com" {
		t.Errorf("email = %q, want lowercase and trimmed", u.Email)
	}
	if u.Name != "King Adesanya" {
		t.Errorf("name = %q, want trimmed", u.Name)
	}

	stored := store.users["king@example.com"]
	if stored.PasswordHash == "correct-horse" || !strings.HasPrefix(stored.PasswordHash, "$2") {
		t.Error("password must be stored as a bcrypt hash, never as plain text")
	}
}

func TestRegister_DuplicateEmail(t *testing.T){
	svc, _ := newTestService()
	in := RegisterInput{Name: "King", Email: "king@example.com", Password: "correct-horse"}

	if _, err := svc.Register(context.Background(), in); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	in.Email = "KING@example.com" // same email, different capitalization
	if _, err := svc.Register(context.Background(), in); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("second Register error = %v, want ErrEmailTaken", err)
	}
}

func TestRegister_PasswordTooLong(t *testing.T){
	svc, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "King", Email: "king@example.com", Password: strings.Repeat("a", 73),
	})
	if !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("error = %v, want ErrPasswordTooLong", err)
	}
}

func TestLogin(t *testing.T){
	svc, _ := newTestService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, RegisterInput{Name: "King", Email: "king@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("setup Register: %v", err)
	}

	tests := []struct{
		name string
		email string
		password string
		wantErr error
	}{
		{"valid credentials", "king@example.com", "correct-horse", nil},
		{"email is case-insensitive", "KING@example.com", "correct-horse", nil},
		{"wrong password", "king@example.com", "wrong-password", ErrInvalidCredentials},
		{"unknown email", "nobody@example.com", "correct-horse", ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.Login(ctx, tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && res.Token == "" {
				t.Error("expected a token on successful login")
			}
			if tt.wantErr != nil && res != nil {
				t.Error("expected a nil result on failure")
			}
		})
	}
}