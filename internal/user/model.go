package user

import(
	"time"

	"github.com/google/uuid"
)

type User struct{
	ID uuid.UUID `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	Currency string `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PasswordHash string `json:"-"`
}