package account

import (
	"time"

	"github.com/google/uuid"
)

const(
	TypeBank = "bank"
	TypeCash = "cash"
	TypeWallet = "wallet"
)

type Account struct{
	ID uuid.UUID `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Currency string `json:"currency"`
	Balance int64 `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID uuid.UUID `json:"-"`
}

func validType(t string) bool{
	return t == TypeBank || t == TypeCash || t == TypeWallet
}