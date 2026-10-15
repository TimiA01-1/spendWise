package alert

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"spendWise/internal/auth/authctx"
	"spendWise/pkg/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

//list handles GET /api/v1/alerts?unread...
func (h *Handler) List(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}

	unreadOnly := false
	if v := c.Query("unread"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "invalid_query", "unread must be true or false")
			return
		}
		unreadOnly = parsed
	}
	limit, err := optionalInt(c, "limit")
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_query", "limit must be a whole number")
		return
	}
	offset, err := optionalInt(c, "offset")
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_query", "offset must be a whole number")
		return
	}

	alerts, err := h.svc.List(c.Request.Context(), userID, unreadOnly, limit, offset)
	if err != nil {
		log.Printf("alert error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.OK(c, http.StatusOK, alerts)
}

// MarkRead handles PATCH /api/v1/alerts/:id/read
func (h *Handler) MarkRead(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "alert id must be a valid UUID")
		return
	}

	if err := h.svc.MarkRead(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "alert_not_found", "alert not found")
			return
		}
		log.Printf("alert error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	c.Status(http.StatusNoContent)
}

func currentUser(c *gin.Context)(uuid.UUID, bool){
	id, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
	}
	return id, ok
}

func optionalInt(c *gin.Context, key string) (int, error) {
	v := c.Query(key)
	if v == "" {
		return 0, nil
	}
	return strconv.Atoi(v)
}