package category

import(
	"time"

	"github.com/google/uuid"
)

const(
	TypeIncome = "income"
	TypeExpense = "expense"
)

// category classifies a transaction. built in categories are shared by every user, custom ones belong to a single user
type Category struct{
	ID uuid.UUID `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	IsDefault bool `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

func validType(t string) bool{
	return t == TypeIncome || t == TypeExpense
}