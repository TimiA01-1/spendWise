package dashboard

import (
	"spendWise/internal/budget"
	"spendWise/internal/transaction"
)

type CurrencySummary struct {
	Currency string `json:"currency"`
	Balance  int64  `json:"current_balance"` 
	Income int64  `json:"income"`          
	Expenses int64  `json:"expenses"`        
	Net int64  `json:"net"`             
}

// dashboard is evrything the home screen needs in one response
type Dashboard struct {
	Month  string `json:"month"` // YYYY-MM
	Summary []CurrencySummary  `json:"summary"`
	RecentTransactions []transaction.Transaction `json:"recent_transactions"`
	Budgets []budget.Budget `json:"budgets"`
	UnreadAlerts int `json:"unread_alerts"`
}