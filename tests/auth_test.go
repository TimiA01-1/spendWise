package tests

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"spendWise/internal/auth"
)

func TestHealth(t *testing.T) {
	r := expect(t, do(t, "GET", "/health", "", nil), http.StatusOK)
	if !strings.Contains(string(r.Body), `"ok"`) {
		t.Errorf("body = %s", r.Body)
	}
}

func TestAuth_RegisterLoginAndProfile(t *testing.T) {
	email := "Mixed.Case-" + uuid.NewString() + "@Example.com"

	reg := expect(t, do(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name": "Ada Lovelace", "email": email, "password": "correct-horse",
	}), http.StatusCreated)

	// The password (or its hash) must never appear in any response.
	for _, forbidden := range []string{"password", "hash", "$2a$"} {
		if strings.Contains(strings.ToLower(string(reg.Body)), forbidden) {
			t.Errorf("register response leaks %q: %s", forbidden, reg.Body)
		}
	}
	var created struct {
		ID       uuid.UUID `json:"id"`
		Email    string    `json:"email"`
		Currency string    `json:"currency"`
	}
	reg.data(t, &created)
	if created.Email != strings.ToLower(email) {
		t.Errorf("email stored as %q, want lowercase %q", created.Email, strings.ToLower(email))
	}
	if created.Currency != "NGN" {
		t.Errorf("default currency = %q, want NGN", created.Currency)
	}

	// Login ignores the case of the email.
	var login struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	expect(t, do(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"email": strings.ToUpper(email), "password": "correct-horse",
	}), http.StatusOK).data(t, &login)
	if login.Token == "" || !login.ExpiresAt.After(time.Now()) {
		t.Fatalf("login = %+v, want a token that expires in the future", login)
	}

	// The token opens the protected profile route.
	var me struct {
		ID uuid.UUID `json:"id"`
	}
	expect(t, do(t, "GET", "/api/v1/users/me", login.Token, nil), http.StatusOK).data(t, &me)
	if me.ID != created.ID {
		t.Errorf("/users/me returned %s, want %s", me.ID, created.ID)
	}
}

func TestAuth_DuplicateEmailIsCaseInsensitive(t *testing.T) {
	u := newUser(t)
	r := expect(t, do(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name": "Someone Else", "email": strings.ToUpper(u.Email), "password": "another-password",
	}), http.StatusConflict)
	if code := r.errorCode(t); code != "email_taken" {
		t.Errorf("code = %q, want email_taken", code)
	}
}

// A wrong password and an unknown email must be indistinguishable, so the API
// can't be used to discover which emails have accounts.
func TestAuth_WrongPasswordAndUnknownEmailLookIdentical(t *testing.T) {
	u := newUser(t)

	wrongPassword := expect(t, do(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"email": u.Email, "password": "definitely-not-it",
	}), http.StatusUnauthorized)
	unknownEmail := expect(t, do(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"email": "nobody-" + uuid.NewString() + "@example.com", "password": u.Password,
	}), http.StatusUnauthorized)

	if !bytes.Equal(wrongPassword.Body, unknownEmail.Body) {
		t.Errorf("responses differ:\n wrong password: %s\n unknown email:  %s", wrongPassword.Body, unknownEmail.Body)
	}
	if code := wrongPassword.errorCode(t); code != "invalid_credentials" {
		t.Errorf("code = %q, want invalid_credentials", code)
	}
}

func TestAuth_RegisterValidation(t *testing.T) {
	long := strings.Repeat("a", 73)
	tests := []struct {
		name string
		body any
		wantStatus int
		wantCode string
		wantFields []string
	}{
		{"short password", map[string]any{"name": "Ada", "email": "a@example.com", "password": "short"},
			422, "validation_failed", []string{"password"}},
		{"password over bcrypt's 72 limit", map[string]any{"name": "Ada", "email": "b@example.com", "password": long},
			422, "validation_failed", []string{"password"}},
		{"bad email", map[string]any{"name": "Ada", "email": "not-an-email", "password": "correct-horse"},
			422, "validation_failed", []string{"email"}},
		{"name too short", map[string]any{"name": "A", "email": "c@example.com", "password": "correct-horse"},
			422, "validation_failed", []string{"name"}},
		{"everything missing reports every field", map[string]any{},
			422, "validation_failed", []string{"name", "email", "password"}},
		{"malformed JSON", `{"name": `, 400, "invalid_body", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := expect(t, do(t, "POST", "/api/v1/auth/register", "", tt.body), tt.wantStatus)
			if code := r.errorCode(t); code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
			fields := r.errorFields(t)
			for _, f := range tt.wantFields {
				if fields[f] == "" {
					t.Errorf("no message for field %q in %v", f, fields)
				}
			}
		})
	}
}

func TestAuth_ProtectedRoutesRejectBadCredentials(t *testing.T) {
	otherSecret, _, _ := auth.NewTokenManager("some-other-secret-that-is-32-characters-long", time.Hour).Generate(uuid.New())
	expired, _, _ := auth.NewTokenManager(testJWTSecret, -time.Hour).Generate(uuid.New())

	headers := map[string]string{
		"no header": "",
		"wrong scheme": "Basic abc123",
		"bearer without a token": "Bearer ",
		"garbage token": "Bearer not.a.jwt",
		"signed with another secret": "Bearer " + otherSecret,
		"expired token": "Bearer " + expired,
	}
	routes := []struct{ method, path string }{
		{"GET", "/api/v1/users/me"},
		{"GET", "/api/v1/accounts"},
		{"POST", "/api/v1/accounts"},
		{"GET", "/api/v1/transactions"},
		{"POST", "/api/v1/transactions"},
		{"GET", "/api/v1/budgets"},
		{"GET", "/api/v1/alerts"},
		{"GET", "/api/v1/dashboard"},
		{"GET", "/api/v1/reports/monthly"},
		{"GET", "/api/v1/recurring-transactions"},
	}
	for name, header := range headers {
		for _, route := range routes {
			t.Run(name+" "+route.method+" "+route.path, func(t *testing.T) {
				r := expect(t, doWithAuth(t, route.method, route.path, header, nil), http.StatusUnauthorized)
				if code := r.errorCode(t); code != "unauthorized" {
					t.Errorf("code = %q, want unauthorized", code)
				}
			})
		}
	}
}

// A perfectly valid token for an account that no longer exists must not work.
func TestAuth_TokenForUnknownUserIsRejectedOnProfile(t *testing.T) {
	ghost, _, _ := auth.NewTokenManager(testJWTSecret, time.Hour).Generate(uuid.New())
	expect(t, do(t, "GET", "/api/v1/users/me", ghost, nil), http.StatusUnauthorized)
}
