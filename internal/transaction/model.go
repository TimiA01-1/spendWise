package transaction

import(
	"time"

	"github.com/google/uuid"
)

const(
	TypeIncome = "income"
	TypeExpense = "expense"
	// MaxAccount caps a single transaction ata 10 billion(in kobo)
	MaxAmount int64 = 1_000_000_000_000
)

// transaction is one income or expense.Amount is always positive, in kobo
type Transaction struct{
	ID uuid.UUID `json:"id"`
	AccountID uuid.UUID `json:"account_id"`
	CategoryID uuid.UUID `json:"category_id"`
	CategoryName string `json:"category_name"`
	Type string `json:"type"`
	Amount int64 `json:"amount"`
	Note string `json:"note"`
	Date string `json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

const dateLayout = "2006-01-02"

func validType(t string) bool{
	return t == TypeIncome || t == TypeExpense
}