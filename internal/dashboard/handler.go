package dashboard

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"spendWise/internal/auth/authctx"
	"spendWise/pkg/httpx"
	"spendWise/pkg/period"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Get handles GET /api/v1/dashboard?month=2026-10 (month is optional).
func (h *Handler) Get(c *gin.Context) {
	userID, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	d, err := h.svc.Get(c.Request.Context(), userID, c.Query("month"))
	switch {
	case errors.Is(err, period.ErrInvalidMonth):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	case err != nil:
		log.Printf("dashboard error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	default:
		httpx.OK(c, http.StatusOK, d)
	}
}