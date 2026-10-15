package alert

import(
	"fmt"
	"time"

	"github.com/google/uuid"

	"spendWise/internal/budget"
)

type Alert struct{
	ID uuid.UUID `json:"id"`
	BudgetID uuid.UUID `json:"budget_id"`
	CategoryName string `json:"category_name"`
	Month string `json:"month"`
	Level string `json:"level"`
	Amount int64 `json:"amount"`
	Spent int64 `json:"spent"`
	PercentUsed int `json:"percent_used"`
	Message string `json:"message"`
	Read bool `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// message builds a short human-readable sentence.
func message(categoryName, level string, percent int) string {
	if level == budget.StatusLimitReached {
		return fmt.Sprintf("%s: you have reached your monthly budget limit (%d%% used)", categoryName, percent)
	}
	return fmt.Sprintf("%s: %d%% of your monthly budget is used", categoryName, percent)
}