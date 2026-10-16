package dashboard

import (
	"reflect"
	"testing"
)

func TestBuildSummary(t *testing.T) {
	balances := []BalanceRow{{"NGN", 17_700_000}, {"USD", 100_000}, {"EUR", 0}}
	totals := []TotalsRow{
		{"NGN", 20_000_000, 2_000_000},
		{"USD", 100_000, 0},
		{"GBP", 0, 5_000}, // activity in a currency with no account balance row
	}

	got := buildSummary(balances, totals)
	want := []CurrencySummary{
		{Currency: "EUR", Balance: 0, Income: 0, Expenses: 0, Net: 0},
		{Currency: "GBP", Balance: 0, Income: 0, Expenses: 5_000, Net: -5_000},
		{Currency: "NGN", Balance: 17_700_000, Income: 20_000_000, Expenses: 2_000_000, Net: 18_000_000},
		{Currency: "USD", Balance: 100_000, Income: 100_000, Expenses: 0, Net: 100_000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSummary =\n %+v\nwant\n %+v", got, want)
	}

	if empty := buildSummary(nil, nil); empty == nil || len(empty) != 0 {
		t.Errorf("no data must give an empty (non-nil) slice, got %#v", empty)
	}
}