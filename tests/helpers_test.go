package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// ---------- making requests ----------

type response struct {
	Status int
	Body []byte
}

// do sends a request to the real router, without a network and returns the status and body
func do(t *testing.T, method, path, token string, body any) response {
	t.Helper()
	return doWithAuth(t, method, path, bearer(token), body)
}

func doWithAuth(t *testing.T, method, path, authorization string, body any) response {
	t.Helper()
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	testHandler.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes()}
}

func bearer(token string) string {
	if token == "" {
		return ""
	}
	return "Bearer " + token
}

// expect fails the test (showing the response body) unless the status matches.
func expect(t *testing.T, r response, status int) response {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status = %d, want %d\nbody: %s", r.Status, status, r.Body)
	}
	return r
}

// data decodes the {"data": ...} envelope into v.
func (r response) data(t *testing.T, v any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("response is not JSON: %v\nbody: %s", err, r.Body)
	}
	if err := json.Unmarshal(env.Data, v); err != nil {
		t.Fatalf("decode data: %v\nbody: %s", err, r.Body)
	}
}

// errorCode returns the machine-readable code of an error response.
func (r response) errorCode(t *testing.T) string {
	t.Helper()
	var env struct {
		Error struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil || env.Error.Code == "" {
		t.Fatalf("not an error response: %s", r.Body)
	}
	return env.Error.Code
}

// errorFields returns the per-field validation messages of a 422 response.
func (r response) errorFields(t *testing.T) map[string]string {
	t.Helper()
	var env struct {
		Error struct {
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("not JSON: %s", r.Body)
	}
	return env.Error.Fields
}

// parallel runs fn(0..n-1) at the same time and waits for all of them.
func parallel(n int, fn func(i int)) {
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			fn(i)
		}(i)
	}
	wg.Wait()
}

// ---------- test users and data ----------

type testUser struct {
	ID uuid.UUID
	Email string
	Password string
	Token string

	openings map[uuid.UUID]int64 // account -> opening balance, for the ledger audit
}

// newUser registers and logs in a brand-new user with a unique email.
func newUser(t *testing.T) *testUser {
	t.Helper()
	u := &testUser{
		Email:    "user-" + uuid.NewString() + "@example.com",
		Password: "correct-horse-battery",
		openings: map[uuid.UUID]int64{},
	}

	var reg struct {
		ID uuid.UUID `json:"id"`
	}
	expect(t, do(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name": "Test User", "email": u.Email, "password": u.Password,
	}), http.StatusCreated).data(t, &reg)
	u.ID = reg.ID

	var login struct {
		Token string `json:"token"`
	}
	expect(t, do(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"email": u.Email, "password": u.Password,
	}), http.StatusOK).data(t, &login)
	u.Token = login.Token
	return u
}

func (u *testUser) do(t *testing.T, method, path string, body any) response {
	t.Helper()
	return do(t, method, path, u.Token, body)
}

type accountJSON struct {
	ID uuid.UUID `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Currency string `json:"currency"`
	Balance  int64 `json:"balance"`
}

// account creates an account through the API and remembers its opening balance.
func (u *testUser) account(t *testing.T, name string, opening int64) uuid.UUID {
	t.Helper()
	var a accountJSON
	expect(t, u.do(t, "POST", "/api/v1/accounts", map[string]any{
		"name": name, "type": "bank", "opening_balance": opening,
	}), http.StatusCreated).data(t, &a)
	u.openings[a.ID] = opening
	return a.ID
}

func (u *testUser) balance(t *testing.T, account uuid.UUID) int64 {
	t.Helper()
	var a accountJSON
	expect(t, u.do(t, "GET", "/api/v1/accounts/"+account.String(), nil), http.StatusOK).data(t, &a)
	return a.Balance
}

type txJSON struct {
	ID uuid.UUID `json:"id"`
	AccountID uuid.UUID `json:"account_id"`
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Type string `json:"type"`
	Amount int64 `json:"amount"`
	Note string  `json:"note"`
	Date string `json:"date"`
}

func txBody(account, category uuid.UUID, typ string, amount int64, date string) map[string]any {
	return map[string]any{
		"account_id": account, "category_id": category,
		"type": typ, "amount": amount, "date": date,
	}
}

// spend records a transaction through the API (expecting success) and returns it.
func (u *testUser) spend(t *testing.T, account, category uuid.UUID, typ string, amount int64, date string) txJSON {
	t.Helper()
	var tx txJSON
	expect(t, u.do(t, "POST", "/api/v1/transactions", txBody(account, category, typ, amount, date)),
		http.StatusCreated).data(t, &tx)
	return tx
}

// assertLedger checks the core promise of the app
func (u *testUser) assertLedger(t *testing.T) {
	t.Helper()
	for account, opening := range u.openings {
		var sum int64
		err := testPool.QueryRow(context.Background(), `
			SELECT COALESCE(SUM(CASE type WHEN 'income' THEN amount ELSE -amount END), 0)::bigint
			FROM transactions WHERE account_id = $1`, account).Scan(&sum)
		if err != nil {
			t.Fatalf("ledger query: %v", err)
		}
		if got, want := u.balance(t, account), opening+sum; got != want {
			t.Errorf("ledger mismatch on account %s: balance %d, but opening %d + transactions %d = %d",
				account, got, opening, sum, want)
		}
	}
}

// categoryID looks up a built-in category (seeded by the migrations).
func categoryID(t *testing.T, name, typ string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM categories WHERE name = $1 AND type = $2 AND user_id IS NULL`, name, typ).Scan(&id)
	if err != nil {
		t.Fatalf("category %q (%s): %v", name, typ, err)
	}
	return id
}
