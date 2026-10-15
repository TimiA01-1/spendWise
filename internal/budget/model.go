package budget

import(
	"time"

	"github.com/google/uuid"
)

// status tells how close spending is to the limit
const(
	StatusOK = "ok"
	StatusWarning = "warning"
	StatusLimitReached = "limit_reached"
)

const(
	DefaultThreshold = 80
	MaxAmount int64 = 1_000_000_000_000

	monthLayout = "2006-01"
)

// budget is a monthly spending limit. with a category, it limits the category; without one, it limits all expenses
// the limit repeats every month
type Budget struct{
	ID uuid.UUID `json:"id"`
	CategoryID *uuid.UUID `json:"category_id"`
	CategoryName string `json:"category_name"`
	Amount int64 `json:"amount"`
	AlertThreshold int `json:"alert_threshold"`
	CreatedAt time.Time `json:"created_at"`
	//calculated for one month
	Month string `json:"month"`
	Spent int64 `json:"spent"`
	Remaining int64 `json:"remaining"`
	PercentUsed int `json:"percent_used"`
	Status string `json:"status"`
}

// Evaluate fills in the calculated fields from Amount, spent and threshold
func (b *Budget) Evaluate() {
	b.Remaining = b.Amount - b.Spent
	b.PercentUsed = int(b.Spent * 100 / b.Amount)

	switch {
	case b.Spent >= b.Amount:
		b.Status = StatusLimitReached
	case b.Spent*100 >= b.Amount*int64(b.AlertThreshold):
		b.Status = StatusWarning
	default:
		b.Status = StatusOK
	}
}

// monthRange returns the first instant of the month and of the next month
func monthRange(month string, now time.Time) (from, to time.Time, label string, err error) {
	if month == "" {
		n := now.UTC()
		from = time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		from, err = time.Parse(monthLayout, month)
		if err != nil {
			return time.Time{}, time.Time{}, "", ErrInvalidMonth
		}
	}
	return from, from.AddDate(0, 1, 0), from.Format(monthLayout), nil
}