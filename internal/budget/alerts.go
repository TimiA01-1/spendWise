package budget

import(
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"spendWise/internal/db"
)

func RecordAlerts(ctx context.Context, q *db.Queries, userID, categoryID uuid.UUID, date time.Time) error{
	date = date.UTC()
	from := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	if _, err := q.LockBudgetsForCategory(ctx, db.LockBudgetsForCategoryParams{
		UserID: userID, CategoryID: categoryID,
	}); err != nil {
		return fmt.Errorf("lock budgets: %w", err)
	}

	rows, err := q.ListAffectedBudgets(ctx, db.ListAffectedBudgetsParams{
		UserID: userID, CategoryID: categoryID, FromDate: from, ToDate: to,
	})
	if err != nil {
		return fmt.Errorf("list affected budgets: %w", err)
	}

	for _, row := range rows{
		b := Budget{Amount: row.Amount, Spent: row.Spent, AlertThreshold: int(row.AlertThreshold)}
		b.Evaluate()
		if b.Status == StatusOK {
			continue
		}

		// b.Status is "warning" or "limit reached"
		if _, err := q.InsertAlert(ctx, db.InsertAlertParams{
			UserID: userID,
			BudgetID: row.ID,
			Month: from,
			Level: b.Status,
			Amount: b.Amount,
			Spent: b.Spent,
			PercentUsed: int32(b.PercentUsed),
		}); err != nil{
			return fmt.Errorf("insert alert: %w", err)
		}
	}
	return nil
}