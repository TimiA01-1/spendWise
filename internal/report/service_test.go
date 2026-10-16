package report

import (
	"testing"

	"github.com/google/uuid"
)

func row(cur, typ, name string, total, count int64) CategoryRow {
	return CategoryRow{Currency: cur, Type: typ, CategoryID: uuid.New(), CategoryName: name, Total: total, Count: count}
}

func TestBuildMonthly(t *testing.T) {
	current := []CategoryRow{
		// rows arrive ordered by currency, type, then biggest total first
		row("NGN", "expense", "Food", 1_500_000, 2),
		row("NGN", "expense", "Entertainment", 1_000_000, 1),
		row("NGN", "expense", "Transport", 500_000, 1),
		row("NGN", "income", "Salary", 20_000_000, 1),
		row("NGN", "income", "Business", 5_000_000, 1),
		row("USD", "expense", "Food", 30_000, 1),
	}
	previous := []ExpenseRow{
		{"NGN", 2_500_000},
		{"EUR", 8_000}, // spent last month, nothing this month
	}

	rep := buildMonthly("2026-10", current, previous)
	if rep.Month != "2026-10" || len(rep.Currencies) != 3 {
		t.Fatalf("month/currencies = %q / %d, want 2026-10 / 3", rep.Month, len(rep.Currencies))
	}
	eur, ngn, usd := rep.Currencies[0], rep.Currencies[1], rep.Currencies[2] // sorted by code
	if eur.Currency != "EUR" || ngn.Currency != "NGN" || usd.Currency != "USD" {
		t.Fatalf("currency order = %s %s %s", eur.Currency, ngn.Currency, usd.Currency)
	}

	// NGN: totals are sums of the category rows.
	if ngn.Income != 25_000_000 || ngn.Expenses != 3_000_000 || ngn.Net != 22_000_000 || ngn.TransactionCount != 6 {
		t.Errorf("NGN totals = %+v", ngn)
	}
	if ngn.PreviousExpenses != 2_500_000 || ngn.ExpensesChange != 500_000 {
		t.Errorf("NGN comparison = prev %d change %d", ngn.PreviousExpenses, ngn.ExpensesChange)
	}
	if ngn.ExpensesChangePercent == nil || *ngn.ExpensesChangePercent != 20 {
		t.Errorf("NGN change percent = %v, want 20", ngn.ExpensesChangePercent)
	}
	// Shares: 1.5M, 1.0M, 0.5M of 3.0M -> 50.0, 33.3 (rounded down), 16.6 (rounded down)
	wantPct := []float64{50, 33.3, 16.6}
	for i, w := range wantPct {
		if got := ngn.SpendingByCategory[i].Percent; got != w {
			t.Errorf("category %d percent = %v, want %v", i, got, w)
		}
	}
	if ngn.IncomeByCategory[0].CategoryName != "Salary" || ngn.IncomeByCategory[0].Percent != 80 {
		t.Errorf("income breakdown = %+v", ngn.IncomeByCategory)
	}

	// USD: no previous month, so the percent change is null (not infinity, not 0).
	if usd.ExpensesChangePercent != nil || usd.ExpensesChange != 30_000 || usd.PreviousExpenses != 0 {
		t.Errorf("USD comparison = %+v", usd)
	}

	// EUR: spending dropped to zero. The row still appears, with a -100% change.
	if eur.Expenses != 0 || eur.PreviousExpenses != 8_000 || eur.ExpensesChange != -8_000 ||
		eur.ExpensesChangePercent == nil || *eur.ExpensesChangePercent != -100 {
		t.Errorf("EUR comparison = %+v", eur)
	}
	if eur.SpendingByCategory == nil || len(eur.SpendingByCategory) != 0 {
		t.Errorf("EUR categories must be an empty non-nil slice, got %#v", eur.SpendingByCategory)
	}
}

func TestBuildMonthly_Empty(t *testing.T) {
	rep := buildMonthly("2026-10", nil, nil)
	if rep.Currencies == nil || len(rep.Currencies) != 0 {
		t.Errorf("no data must give an empty non-nil slice, got %#v", rep.Currencies)
	}
}

func TestBuildCategories(t *testing.T) {
	rows := []CategoryRow{
		row("NGN", "expense", "Food", 3_500_000, 4),
		row("NGN", "expense", "Transport", 1_000_000, 2),
		row("NGN", "income", "Salary", 20_000_000, 1), // wrong type for this report, must be ignored
		row("USD", "expense", "Food", 10_000, 1),
	}

	rep := buildCategories("2026-09-01", "2026-10-31", "expense", rows)
	if len(rep.Currencies) != 2 || rep.Type != "expense" {
		t.Fatalf("report = %+v", rep)
	}
	ngn := rep.Currencies[0]
	if ngn.Currency != "NGN" || ngn.Total != 4_500_000 || len(ngn.Categories) != 2 {
		t.Errorf("NGN = %+v", ngn)
	}
	if ngn.Categories[0].Percent != 77.7 || ngn.Categories[1].Percent != 22.2 {
		t.Errorf("NGN percents = %v, %v (want 77.7 and 22.2, rounded down)", ngn.Categories[0].Percent, ngn.Categories[1].Percent)
	}
}

func TestPercent(t *testing.T) {
	tests := []struct {
		part, whole int64
		want        float64
	}{
		{0, 100, 0},
		{50, 100, 50},
		{1, 3, 33.3},
		{2, 3, 66.6},
		{100, 100, 100},
		{150, 100, 150},
		{-50, 100, -50},
		{5, 0, 0}, // no division by zero
	}
	for _, tt := range tests {
		if got := percent(tt.part, tt.whole); got != tt.want {
			t.Errorf("percent(%d, %d) = %v, want %v", tt.part, tt.whole, got, tt.want)
		}
	}
}