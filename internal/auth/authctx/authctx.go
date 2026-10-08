package authctx

import(
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const key = "auth.user_id"

// set is called by auth middleware after a token is validated
func Set(c *gin.Context, id uuid.UUID){ c.Set(key, id) }

// userID returns to the authenticated user's id
func UserID(c *gin.Context)(uuid.UUID, bool){
	v, exists := c.Get(key)
	if !exists{
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}