package tests

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// Walks one transaction through every kind of edit and checks the balances
// after each step: the balance must always equal opening + transaction history.
func TestTransactions_BalanceStaysCorrectThroughEveryEdit(t *testing.T) {
	u := newUser(t)
	a := u.account(t, "Account A", 5_000_000)
	b := u.account(t, "Account B", 0)
	food, salary := categoryID(t, "Food", "expense"), categoryID(t, "Salary", "income")
	step := func(name string, wantA, wantB int64) {
		t.Helper()
		if gotA, gotB := u.balance(t, a), u.balance(t, b); gotA != wantA || gotB != wantB {
			t.Fatalf("after %s: A=%d B=%d, want A=%d B=%d", name, gotA, gotB, wantA, wantB)
		}
		u.assertLedger(t)
	}

	expense := u.spend(t, a, food, "expense", 1_500_000, "2025-03-10") // a ₦15,000 food expense
	step("expense of 1,500,000", 3_500_000, 0)

	income := u.spend(t, a, salary, "income", 200_000, "2025-03-11")
	step("income of 200,000", 3_700_000, 0)

	path := "/api/v1/transactions/" + expense.ID.String()

	expect(t, u.do(t, "PATCH", path, map[string]any{"amount": 2_000_000}), http.StatusOK)
	step("raising the expense to 2,000,000", 3_200_000, 0)

	expect(t, u.do(t, "PATCH", path, map[string]any{"note": "only a note"}), http.StatusOK)
	step("changing only the note", 3_200_000, 0)

	// An income can't keep an expense category: refused, nothing moves.
	expect(t, u.do(t, "PATCH", path, map[string]any{"type": "income"}), http.StatusUnprocessableEntity)
	step("a refused type change", 3_200_000, 0)

	// Type AND category together: the 2,000,000 expense is undone and a 2,000,000 income added.
	expect(t, u.do(t, "PATCH", path, map[string]any{"type": "income", "category_id": salary}), http.StatusOK)
	step("expense becoming income", 7_200_000, 0)

	expect(t, u.do(t, "PATCH", path, map[string]any{"account_id": b}), http.StatusOK)
	step("moving to account B", 5_200_000, 2_000_000)

	expect(t, u.do(t, "DELETE", path, nil), http.StatusNoContent)
	step("deleting the moved income", 5_200_000, 0)

	expect(t, u.do(t, "DELETE", "/api/v1/transactions/"+income.ID.String(), nil), http.StatusNoContent)
	step("deleting the other income", 5_000_000, 0) // back to the opening balance
}

func TestTransactions_Validation(t *testing.T) {
	u := newUser(t)
	stranger := newUser(t)
	acc := u.account(t, "Main", 100_000)
	strangerAcc := stranger.account(t, "Theirs", 100)
	food, salary := categoryID(t, "Food", "expense"), categoryID(t, "Salary", "income")

	// a custom category that belongs to the stranger
	var strangersCategory uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`INSERT INTO categories (user_id, name, type) VALUES ($1, 'Private', 'expense') RETURNING id`, stranger.ID).Scan(&strangersCategory)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		body       any
		wantStatus int
		wantCode   string
	}{
		{"category type doesn't match", txBody(acc, salary, "expense", 100, "2025-03-10"), 422, "validation_failed"},
		{"someone else's account", txBody(strangerAcc, food, "expense", 100, "2025-03-10"), 422, "invalid_account"},
		{"unknown account", txBody(uuid.New(), food, "expense", 100, "2025-03-10"), 422, "invalid_account"},
		{"someone else's custom category", txBody(acc, strangersCategory, "expense", 100, "2025-03-10"), 422, "invalid_category"},
		{"amount zero", txBody(acc, food, "expense", 0, "2025-03-10"), 422, "validation_failed"},
		{"amount over the cap", txBody(acc, food, "expense", 1_000_000_000_001, "2025-03-10"), 422, "validation_failed"},
		{"decimal amount (money is whole kobo)", `{"amount": 10.5}`, 400, "invalid_body"},
		{"bad type", txBody(acc, food, "transfer", 100, "2025-03-10"), 422, "validation_failed"},
		{"bad date format", txBody(acc, food, "expense", 100, "10/03/2025"), 422, "validation_failed"},
		{"ids that aren't UUIDs", map[string]any{"account_id": "nope", "category_id": "x", "type": "expense", "amount": 1}, 422, "validation_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := expect(t, u.do(t, "POST", "/api/v1/transactions", tt.body), tt.wantStatus)
			if code := r.errorCode(t); code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}

	// Not one of those refused requests may have changed anything.
	if got := u.balance(t, acc); got != 100_000 {
		t.Errorf("balance = %d after only failed requests, want 100000", got)
	}
	if got := stranger.balance(t, strangerAcc); got != 100 {
		t.Errorf("the stranger's balance changed to %d", got)
	}
	var list []txJSON
	expect(t, u.do(t, "GET", "/api/v1/transactions", nil), http.StatusOK).data(t, &list)
	if len(list) != 0 {
		t.Errorf("%d transactions were saved by failed requests", len(list))
	}
}

func TestTransactions_Ownership(t *testing.T) {
	alice, bob := newUser(t), newUser(t)
	aliceAcc := alice.account(t, "Alice", 50_000)
	food := categoryID(t, "Food", "expense")
	tx := alice.spend(t, aliceAcc, food, "expense", 1_000, "2025-03-10")
	path := "/api/v1/transactions/" + tx.ID.String()

	var list []txJSON
	expect(t, bob.do(t, "GET", "/api/v1/transactions", nil), http.StatusOK).data(t, &list)
	if len(list) != 0 {
		t.Errorf("bob sees %d of alice's transactions", len(list))
	}

	for _, c := range []struct {
		method string
		body   any
	}{{"GET", nil}, {"PATCH", map[string]any{"note": "hacked"}}, {"DELETE", nil}} {
		r := expect(t, bob.do(t, c.method, path, c.body), http.StatusNotFound)
		if code := r.errorCode(t); code != "transaction_not_found" {
			t.Errorf("%s: code = %q", c.method, code)
		}
	}

	// bob can't spend from alice's account either
	expect(t, bob.do(t, "POST", "/api/v1/transactions", txBody(aliceAcc, food, "expense", 1, "2025-03-10")), http.StatusUnprocessableEntity)

	// alice's data is exactly as she left it
	if got := alice.balance(t, aliceAcc); got != 49_000 {
		t.Errorf("alice's balance = %d, want 49000", got)
	}
	var still txJSON
	expect(t, alice.do(t, "GET", path, nil), http.StatusOK).data(t, &still)
	if still.Note != "" || still.Amount != 1_000 {
		t.Errorf("alice's transaction changed: %+v", still)
	}
	alice.assertLedger(t)
}

func TestTransactions_NotFoundAndBadIDs(t *testing.T) {
	u := newUser(t)
	expect(t, u.do(t, "GET", "/api/v1/transactions/"+uuid.NewString(), nil), http.StatusNotFound)
	expect(t, u.do(t, "GET", "/api/v1/transactions/abc", nil), http.StatusBadRequest)
	expect(t, u.do(t, "PATCH", "/api/v1/transactions/"+uuid.NewString(), map[string]any{"note": "x"}), http.StatusNotFound)
	expect(t, u.do(t, "DELETE", "/api/v1/transactions/"+uuid.NewString(), nil), http.StatusNotFound)
}

func TestTransactions_ListFilters(t *testing.T) {
	u := newUser(t)
	acc1, acc2 := u.account(t, "One", 0), u.account(t, "Two", 0)
	food, transport, salary := categoryID(t, "Food", "expense"), categoryID(t, "Transport", "expense"), categoryID(t, "Salary", "income")

	u.spend(t, acc1, salary, "income", 9_000, "2025-01-05")
	u.spend(t, acc1, food, "expense", 100, "2025-01-10")
	u.spend(t, acc1, food, "expense", 200, "2025-02-10")
	u.spend(t, acc2, transport, "expense", 300, "2025-02-11")
	u.spend(t, acc2, food, "expense", 400, "2025-03-12")

	count := func(query string) int {
		t.Helper()
		var list []txJSON
		expect(t, u.do(t, "GET", "/api/v1/transactions"+query, nil), http.StatusOK).data(t, &list)
		return len(list)
	}

	tests := []struct {
		query string
		want  int
	}{
		{"", 5},
		{"?type=income", 1},
		{"?type=expense", 4},
		{"?account_id=" + acc1.String(), 3},
		{"?account_id=" + acc2.String(), 2},
		{"?category_id=" + food.String(), 3},
		{"?from=2025-02-01&to=2025-02-28", 2},
		{"?from=2025-02-10&to=2025-02-10", 1}, // both ends inclusive
		{"?limit=2", 2},
		{"?limit=2&offset=4", 1},
		{"?limit=1000", 5}, // capped, not an error
		{"?offset=100", 0},
		{"?type=expense&account_id=" + acc2.String() + "&from=2025-03-01", 1},
	}
	for _, tt := range tests {
		if got := count(tt.query); got != tt.want {
			t.Errorf("GET /transactions%s returned %d, want %d", tt.query, got, tt.want)
		}
	}

	// newest first
	var all []txJSON
	expect(t, u.do(t, "GET", "/api/v1/transactions", nil), http.StatusOK).data(t, &all)
	if all[0].Date != "2025-03-12" || all[len(all)-1].Date != "2025-01-05" {
		t.Errorf("order = %s ... %s, want newest first", all[0].Date, all[len(all)-1].Date)
	}

	// bad filters
	expect(t, u.do(t, "GET", "/api/v1/transactions?account_id=oops", nil), http.StatusBadRequest)
	expect(t, u.do(t, "GET", "/api/v1/transactions?from=2025-13-01", nil), http.StatusBadRequest)
	expect(t, u.do(t, "GET", "/api/v1/transactions?limit=x", nil), http.StatusBadRequest)
	expect(t, u.do(t, "GET", "/api/v1/transactions?type=savings", nil), http.StatusUnprocessableEntity)
	expect(t, u.do(t, "GET", "/api/v1/transactions?from=2025-12-01&to=2025-01-01", nil), http.StatusUnprocessableEntity)
}

// 25 bad requests hit one account at the same time and everyone must count
func TestTransactions_ConcurrentSpendingNeverLosesMoney(t *testing.T) {
	u := newUser(t)
	acc := u.account(t, "Busy", 1_000_000)
	food := categoryID(t, "Food", "expense")

	statuses := make([]int, 25)
	parallel(len(statuses), func(i int) {
		statuses[i] = do(t, "POST", "/api/v1/transactions", u.Token,
			txBody(acc, food, "expense", 1_000, "2025-03-10")).Status
	})

	for i, s := range statuses {
		if s != http.StatusCreated {
			t.Errorf("request %d returned %d, want 201", i, s)
		}
	}
	if got := u.balance(t, acc); got != 1_000_000-25*1_000 {
		t.Errorf("balance = %d, want %d (some deductions were lost)", got, 1_000_000-25_000)
	}
	var list []txJSON
	expect(t, u.do(t, "GET", "/api/v1/transactions?limit=100", nil), http.StatusOK).data(t, &list)
	if len(list) != 25 {
		t.Errorf("%d transactions stored, want 25", len(list))
	}
	u.assertLedger(t)
}


func TestTransactions_ConcurrentMovesInBothDirections(t *testing.T) {
	u := newUser(t)
	a, b := u.account(t, "Account A", 1_000_000), u.account(t, "Account B", 1_000_000)
	food := categoryID(t, "Food", "expense")

	var fromA, fromB []uuid.UUID
	for i := 1; i <= 10; i++ {
		fromA = append(fromA, u.spend(t, a, food, "expense", int64(i)*1_000, "2025-03-10").ID)
		fromB = append(fromB, u.spend(t, b, food, "expense", int64(i)*1_000, "2025-03-10").ID)
	}
	totalBefore := u.balance(t, a) + u.balance(t, b)

	statuses := make([]int, 20)
	parallel(len(statuses), func(i int) {
		id, target := fromA[i%10], b // first ten move A -> B ...
		if i >= 10 {
			id, target = fromB[i-10], a // ... the other ten move B -> A, at the same time
		}
		statuses[i] = do(t, "PATCH", "/api/v1/transactions/"+id.String(), u.Token, map[string]any{"account_id": target}).Status
	})

	for i, s := range statuses {
		if s != http.StatusOK {
			t.Errorf("move %d returned %d, want 200 (a 500 here usually means a deadlock)", i, s)
		}
	}
	if total := u.balance(t, a) + u.balance(t, b); total != totalBefore {
		t.Errorf("money changed during the moves: total %d, was %d", total, totalBefore)
	}
	u.assertLedger(t)
}
