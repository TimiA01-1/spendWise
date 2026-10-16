package report

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"spendWise/pkg/period"
)

const dateLayout = "2006-01-02"

var (
	ErrInvalidDate  = errors.New("from and to are required and must look like 2026-10-31")
	ErrInvalidRange = errors.New("'from' date must not be after 'to' date")
	ErrInvalidType  = errors.New("type must be expense or income")
)

type Service struct {
	repo Repository
	now  func() time.Time // injectable for tests
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Monthly builds the report for a month ("" = the current month).
func (s *Service) Monthly(ctx context.Context, userID uuid.UUID, month string) (MonthlyReport, error) {
	from, to, label, err := period.Month(month, s.now())
	if err != nil {
		return MonthlyReport{}, err
	}
	prevFrom := from.AddDate(0, -1, 0)

	current, previous, err := s.repo.Monthly(ctx, userID, from, to, prevFrom, from)
	if err != nil {
		return MonthlyReport{}, fmt.Errorf("monthly report: %w", err)
	}
	return buildMonthly(label, current, previous), nil
}

func (s *Service) Categories(ctx context.Context, userID uuid.UUID, fromStr, toStr, typ string) (CategoryReport, error) {
	if typ == "" {
		typ = "expense"
	}
	if typ != "expense" && typ != "income" {
		return CategoryReport{}, ErrInvalidType
	}
	from, err := time.Parse(dateLayout, fromStr)
	if err != nil {
		return CategoryReport{}, ErrInvalidDate
	}
	to, err := time.Parse(dateLayout, toStr)
	if err != nil {
		return CategoryReport{}, ErrInvalidDate
	}
	if from.After(to) {
		return CategoryReport{}, ErrInvalidRange
	}

	// The database range is half-open (< to), so add a day to include `to`.
	rows, err := s.repo.Categories(ctx, userID, from, to.AddDate(0, 0, 1))
	if err != nil {
		return CategoryReport{}, fmt.Errorf("category report: %w", err)
	}
	return buildCategories(fromStr, toStr, typ, rows), nil
}

// buildMonthly turns raw rows into the report
func buildMonthly(month string, current []CategoryRow, previous []ExpenseRow) MonthlyReport {
	byCurrency := map[string]*CurrencyReport{}
	get := func(cur string) *CurrencyReport {
		if cr, ok := byCurrency[cur]; ok {
			return cr
		}
		cr := &CurrencyReport{
			Currency: cur,
			SpendingByCategory: []CategoryTotal{},
			IncomeByCategory: []CategoryTotal{},
		}
		byCurrency[cur] = cr
		return cr
	}

	for _, row := range current {
		cr := get(row.Currency)
		ct := CategoryTotal{
			CategoryID: row.CategoryID,
			CategoryName: row.CategoryName,
			Total: row.Total,
			Transactions: int(row.Count),
		}
		cr.TransactionCount += int(row.Count)
		if row.Type == "expense" {
			cr.Expenses += row.Total
			cr.SpendingByCategory = append(cr.SpendingByCategory, ct)
		} else {
			cr.Income += row.Total
			cr.IncomeByCategory = append(cr.IncomeByCategory, ct)
		}
	}

	// A currency that had spending last month but nothing this month still gets
	// a row, so the drop to zero is visible.
	for _, p := range previous {
		get(p.Currency).PreviousExpenses = p.Expenses
	}

	out := make([]CurrencyReport, 0, len(byCurrency))
	for _, cr := range byCurrency {
		cr.Net = cr.Income - cr.Expenses
		cr.ExpensesChange = cr.Expenses - cr.PreviousExpenses
		if cr.PreviousExpenses > 0 {
			p := percent(cr.ExpensesChange, cr.PreviousExpenses)
			cr.ExpensesChangePercent = &p
		}
		setPercents(cr.SpendingByCategory, cr.Expenses)
		setPercents(cr.IncomeByCategory, cr.Income)
		out = append(out, *cr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return MonthlyReport{Month: month, Currencies: out}
}

// buildCategories keeps only rows of one type and groups them by currency.
func buildCategories(from, to, typ string, rows []CategoryRow) CategoryReport {
	byCurrency := map[string]*CategoryCurrency{}
	for _, row := range rows {
		if row.Type != typ {
			continue
		}
		cc, ok := byCurrency[row.Currency]
		if !ok {
			cc = &CategoryCurrency{Currency: row.Currency, Categories: []CategoryTotal{}}
			byCurrency[row.Currency] = cc
		}
		cc.Total += row.Total
		cc.Categories = append(cc.Categories, CategoryTotal{
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			Total:        row.Total,
			Transactions: int(row.Count),
		})
	}

	out := make([]CategoryCurrency, 0, len(byCurrency))
	for _, cc := range byCurrency {
		setPercents(cc.Categories, cc.Total)
		out = append(out, *cc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return CategoryReport{From: from, To: to, Type: typ, Currencies: out}
}

func setPercents(cats []CategoryTotal, whole int64) {
	for i := range cats {
		cats[i].Percent = percent(cats[i].Total, whole)
	}
}

func percent(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part*1000/whole) / 10
}
