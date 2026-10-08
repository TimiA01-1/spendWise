package auth

import(
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"spendWise/internal/auth/authctx"
	"spendWise/pkg/httpx"
)

// middleware rejects request without a valid "Authorization: bearer <jwt>" header and stores the users ID for handlers
func Middleware(tokens *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, found := strings.Cut(c.GetHeader("Authorization"), " ")
		token = strings.TrimSpace(token)
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
			httpx.Error(c, http.StatusUnauthorized, "unauthorized", "missing or malformed Authorization header")
			return
		}

		id, err := tokens.Parse(token)
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}

		authctx.Set(c, id)
		c.Next()
	}
}