package user

import(
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"spendWise/internal/auth/authctx"
	"spendWise/pkg/httpx"
)

type Handler struct{
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Me handles GET /api/v1/users/me: the authenticated user's profile.
func (h *Handler) Me(c *gin.Context) {
	id, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	u, err := h.svc.GetProfile(c.Request.Context(), id)
	switch {
	case errors.Is(err, ErrNotFound):
		// Valid token, but the account no longer exists.
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
	case err != nil:
		log.Printf("get profile failed: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	default:
		httpx.OK(c, http.StatusOK, u)
	}
}