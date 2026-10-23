package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestAccounts_CRUD(t *testing.T) {
	u := newUser(t)

	// create: defaults to NGN, opening balance is kept exactly (in kobo)
	var acc accountJSON
	expect(t, u.do(t, "POST", "/api/v1/accounts", map[string]any{
		"name": "GTBank Savings", "type": "bank", "opening_balance": 5_000_000,
	}), http.StatusCreated).data(t, &acc)
	if acc.Balance != 5_000_000 || acc.Currency != "NGN" || acc.Name != "GTBank Savings" {
		t.Fatalf("created account = %+v", acc)
	}
	path := "/api/v1/accounts/" + acc.ID.String()

	// a second one with no opening balance
	var cash accountJSON
	expect(t, u.do(t, "POST", "/api/v1/accounts", map[string]any{"name": "Wallet cash", "type": "cash"}),
		http.StatusCreated).data(t, &cash)
	if cash.Balance != 0 {
		t.Errorf("default opening balance = %d, want 0", cash.Balance)
	}

	// list
	var list []accountJSON
	expect(t, u.do(t, "GET", "/api/v1/accounts", nil), http.StatusOK).data(t, &list)
	if len(list) != 2 {
		t.Errorf("list has %d accounts, want 2", len(list))
	}

	// rename: the balance must not change
	var renamed accountJSON
	expect(t, u.do(t, "PATCH", path, map[string]any{"name": "GTBank Main"}), http.StatusOK).data(t, &renamed)
	if renamed.Name != "GTBank Main" || renamed.Balance != 5_000_000 || renamed.Type != "bank" {
		t.Errorf("after rename = %+v", renamed)
	}

	// the balance is not editable: a PATCH with only "balance" is an empty update
	expect(t, u.do(t, "PATCH", path, map[string]any{"balance": 999_999_999}), http.StatusUnprocessableEntity)
	if got := u.balance(t, acc.ID); got != 5_000_000 {
		t.Errorf("balance = %d after trying to PATCH it, want 5000000", got)
	}

	// delete (no history yet), then it's gone
	expect(t, u.do(t, "DELETE", "/api/v1/accounts/"+cash.ID.String(), nil), http.StatusNoContent)
	r := expect(t, u.do(t, "GET", "/api/v1/accounts/"+cash.ID.String(), nil), http.StatusNotFound)
	if code := r.errorCode(t); code != "account_not_found" {
		t.Errorf("code = %q", code)
	}
	expect(t, u.do(t, "DELETE", "/api/v1/accounts/"+cash.ID.String(), nil), http.StatusNotFound)
}

func TestAccounts_Validation(t *testing.T) {
	u := newUser(t)
	u.account(t, "Existing", 0)

	tests := []struct {
		name string
		body any
		wantStatus int
		wantCode string
	}{
		{"duplicate name, different case", map[string]any{"name": "EXISTING", "type": "bank"}, 409, "account_name_taken"},
		{"invalid type", map[string]any{"name": "Stash", "type": "crypto"}, 422, "validation_failed"},
		{"negative opening balance", map[string]any{"name": "Stash", "type": "bank", "opening_balance": -1}, 422, "validation_failed"},
		{"missing name", map[string]any{"type": "bank"}, 422, "validation_failed"},
		{"decimal balance (money is whole kobo)", `{"name":"Stash","type":"bank","opening_balance":10.5}`, 400, "invalid_body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := expect(t, u.do(t, "POST", "/api/v1/accounts", tt.body), tt.wantStatus)
			if code := r.errorCode(t); code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}

	t.Run("invalid id in the path", func(t *testing.T) {
		expect(t, u.do(t, "GET", "/api/v1/accounts/abc", nil), http.StatusBadRequest)
	})
	t.Run("unknown id", func(t *testing.T) {
		expect(t, u.do(t, "GET", "/api/v1/accounts/"+uuid.NewString(), nil), http.StatusNotFound)
	})
}

// The ownership rule: one user can never see or change another user's account,
// and asking about it looks exactly like asking about one that doesn't exist.
func TestAccounts_Ownership(t *testing.T) {
	alice, bob := newUser(t), newUser(t)
	acc := alice.account(t, "Alice's savings", 1_000_000)
	path := "/api/v1/accounts/" + acc.String()

	// Bob sees an empty list
	var list []accountJSON
	expect(t, bob.do(t, "GET", "/api/v1/accounts", nil), http.StatusOK).data(t, &list)
	if len(list) != 0 {
		t.Errorf("bob sees %d of alice's accounts", len(list))
	}

	// and gets "not found" for everything else, the same answer an unknown id gets
	stranger := "/api/v1/accounts/" + uuid.NewString()
	for _, c := range []struct{ method, path string }{{"GET", path}, {"PATCH", path}, {"DELETE", path}} {
		var body any
		if c.method == "PATCH" {
			body = map[string]any{"name": "hacked"}
		}
		mine := expect(t, bob.do(t, c.method, c.path, body), http.StatusNotFound)
		unknown := expect(t, bob.do(t, c.method, stranger, body), http.StatusNotFound)
		if string(mine.Body) != string(unknown.Body) {
			t.Errorf("%s: someone else's account and a missing one look different:\n %s\n %s", c.method, mine.Body, unknown.Body)
		}
	}

	// Alice's account is untouched
	var after accountJSON
	expect(t, alice.do(t, "GET", path, nil), http.StatusOK).data(t, &after)
	if after.Name != "Alice's savings" || after.Balance != 1_000_000 {
		t.Errorf("alice's account changed: %+v", after)
	}
}

func TestAccounts_CannotBeDeletedWhileTheyHaveHistory(t *testing.T) {
	u := newUser(t)
	acc := u.account(t, "Main", 100_000)
	tx := u.spend(t, acc, categoryID(t, "Food", "expense"), "expense", 1_000, "2025-03-10")
	path := "/api/v1/accounts/" + acc.String()

	r := expect(t, u.do(t, "DELETE", path, nil), http.StatusConflict)
	if code := r.errorCode(t); code != "account_has_transactions" {
		t.Errorf("code = %q, want account_has_transactions", code)
	}
	if got := u.balance(t, acc); got != 99_000 {
		t.Errorf("balance = %d after a refused delete, want 99000", got)
	}

	// once the history is removed, the account can go
	expect(t, u.do(t, "DELETE", "/api/v1/transactions/"+tx.ID.String(), nil), http.StatusNoContent)
	expect(t, u.do(t, "DELETE", path, nil), http.StatusNoContent)
}
