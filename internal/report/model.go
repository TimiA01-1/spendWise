package report

import "github.com/google/uuid"

// CategoryTotal is one row of a category breakdown.
type CategoryTotal struct {
	CategoryID uuid.UUID `json:"category_id"`
	CategoryName string `json:"category_name"`
	Total  int64 `json:"total"`    
	Transactions int `json:"transactions"` 
	Percent float64 `json:"percent"`      
}

// CurrencyReport is one month's picture for one currency.
type CurrencyReport struct {
	Currency string `json:"currency"`
	Income int64  `json:"income"`
	Expenses int64  `json:"expenses"`
	Net int64  `json:"net"` // income - expenses
	TransactionCount int `json:"transaction_count"`

	// Compared with the month before.
	PreviousExpenses int64 `json:"previous_expenses"`
	ExpensesChange int64 `json:"expenses_change"`         // this month - last month
	ExpensesChangePercent *float64 `json:"expenses_change_percent"` // null when last month had no spending

	SpendingByCategory []CategoryTotal `json:"spending_by_category"` // biggest first
	IncomeByCategory   []CategoryTotal `json:"income_by_category"`   // biggest first
}

// MonthlyReport is the response of GET /reports/monthly.
type MonthlyReport struct {
	Month string `json:"month"` // YYYY-MM
	Currencies []CurrencyReport `json:"currencies"`
}

// CategoryCurrency is one currency's category breakdown over a date range.
type CategoryCurrency struct {
	Currency string `json:"currency"`
	Total int64  `json:"total"`
	Categories []CategoryTotal `json:"categories"` // biggest first
}

// CategoryReport is the response of GET /reports/categories.
type CategoryReport struct {
	From string `json:"from"` // YYYY-MM-DD, inclusive
	To string `json:"to"`   // YYYY-MM-DD, inclusive
	Type string  `json:"type"` // "expense" or "income"
	Currencies []CategoryCurrency `json:"currencies"`
}