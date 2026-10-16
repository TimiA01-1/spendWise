package report

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

// Monthly handles GET /api/v1/reports/monthly?month=2026-10 (month is optional).
func (h *Handler) Monthly(c *gin.Context) {
	userID, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	r, err := h.svc.Monthly(c.Request.Context(), userID, c.Query("month"))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, r)
}

// Categories handles GET /api/v1/reports/categories?from=2026-01-01&to=2026-03-31&type=expense
func (h *Handler) Categories(c *gin.Context) {
	userID, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	r, err := h.svc.Categories(c.Request.Context(), userID, c.Query("from"), c.Query("to"), c.Query("type"))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, r)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, period.ErrInvalidMonth), errors.Is(err, ErrInvalidDate),
		errors.Is(err, ErrInvalidRange), errors.Is(err, ErrInvalidType):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		log.Printf("report error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}