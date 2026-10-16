package dashboard

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"spendWise/pkg/period"
)

const recentLimit = 10

type Service struct {
	repo Repository
	now  func() time.Time // injectable for tests
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Get builds the dashboard for a month ("" = the current month).
func (s *Service) Get(ctx context.Context, userID uuid.UUID, month string) (Dashboard, error) {
	from, to, label, err := period.Month(month, s.now())
	if err != nil {
		return Dashboard{}, err
	}

	snap, err := s.repo.Load(ctx, userID, from, to, recentLimit)
	if err != nil {
		return Dashboard{}, fmt.Errorf("load dashboard: %w", err)
	}

	for i := range snap.Budgets {
		snap.Budgets[i].Month = label
		snap.Budgets[i].Evaluate()
	}

	return Dashboard{
		Month:              label,
		Summary:            buildSummary(snap.Balances, snap.Totals),
		RecentTransactions: snap.Recent,  // never nil: the repository always makes a slice
		Budgets:            snap.Budgets, // never nil either, so JSON shows [] not null
		UnreadAlerts:       snap.UnreadAlerts,
	}, nil
}


// buildSummary merges balances and monthly totals into one row per currency.
// A currency appears if the user has an account in it OR had activity in it.
func buildSummary(balances []BalanceRow, totals []TotalsRow) []CurrencySummary {
	byCurrency := map[string]*CurrencySummary{}
	get := func(cur string) *CurrencySummary {
		if cs, ok := byCurrency[cur]; ok {
			return cs
		}
		cs := &CurrencySummary{Currency: cur}
		byCurrency[cur] = cs
		return cs
	}

	for _, b := range balances {
		get(b.Currency).Balance = b.Balance
	}
	for _, t := range totals {
		cs := get(t.Currency)
		cs.Income = t.Income
		cs.Expenses = t.Expenses
	}

	out := make([]CurrencySummary, 0, len(byCurrency))
	for _, cs := range byCurrency {
		cs.Net = cs.Income - cs.Expenses
		out = append(out, *cs)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}