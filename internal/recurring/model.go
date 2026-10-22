// package recurring handles rules like "rent, 150, 000, on the first of every month" and the background worker that turns them into real transactions
package recurring

import (
	"time"

	"github.com/google/uuid"
)

const (
	FreqDaily = "daily"
	FreqWeekly = "weekly"
	FreqMonthly = "monthly"
	FreqYearly = "yearly"

	StatusActive = "active"
	StatusPaused = "paused"
	StatusFinished = "finished"

	dateLayout = "2006-01-02"
)

func validFrequency(f string) bool {
	return f == FreqDaily || f == FreqWeekly || f == FreqMonthly || f == FreqYearly
}

// Rule is a recurring-transaction rule. Dates are calendar dates (UTC).
type Rule struct {
	ID uuid.UUID
	AccountID uuid.UUID
	CategoryID   uuid.UUID
	CategoryName string
	Type string
	Amount int64
	Note string
	Frequency string
	StartDate time.Time
	EndDate *time.Time // nil = never ends
	NextIndex int  // which occurrence, counted from StartDate, is due next
	NextRunOn time.Time  // the date of that occurrence
	Status string
	Runs int // how many transactions this rule has created so far
	LastError string
	CreatedAt time.Time
}

// RuleView is how a Rule looks in API responses.
type RuleView struct {
	ID uuid.UUID `json:"id"`
	AccountID uuid.UUID `json:"account_id"`
	CategoryID uuid.UUID `json:"category_id"`
	CategoryName string `json:"category_name"`
	Type string `json:"type"`
	Amount int64 `json:"amount"`
	Note string `json:"note"`
	Frequency string  `json:"frequency"`
	StartDate string `json:"start_date"`
	EndDate *string `json:"end_date"` // null = never ends
	NextRunOn *string `json:"next_run_on"` // null once the rule has finished
	Status string `json:"status"`
	Runs int `json:"runs"`
	LastError string `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (r Rule) View() RuleView {
	v := RuleView{
		ID: r.ID,
		AccountID: r.AccountID,
		CategoryID: r.CategoryID,
		CategoryName: r.CategoryName,
		Type: r.Type,
		Amount: r.Amount,
		Note: r.Note,
		Frequency: r.Frequency,
		StartDate: r.StartDate.Format(dateLayout),
		Status: r.Status,
		Runs: r.Runs,
		LastError: r.LastError,
		CreatedAt: r.CreatedAt,
	}
	if r.EndDate != nil{
		s := r.EndDate.Format(dateLayout)
		v.EndDate = &s
	}
	if r.Status != StatusFinished {
		s := r.NextRunOn.Format(dateLayout)
		v.NextRunOn = &s
	}
	return v
}