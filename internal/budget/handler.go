package budget

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"spendWise/internal/auth/authctx"
	"spendWise/pkg/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createRequest struct {
	CategoryID *string `json:"category_id" binding:"omitempty,uuid"` // omit for an overall budget
	Amount int64 `json:"amount" binding:"required,gt=0"`
	AlertThreshold *int `json:"alert_threshold" binding:"omitempty,gte=1,lte=100"`
}

type updateRequest struct {
	Amount *int64 `json:"amount" binding:"omitempty,gt=0"`
	AlertThreshold *int `json:"alert_threshold" binding:"omitempty,gte=1,lte=100"`
}

// Create handles POST /api/v1/budgets
func (h *Handler) Create(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}

	in := CreateInput{Amount: req.Amount, Threshold: req.AlertThreshold}
	if req.CategoryID != nil {
		in.CategoryID, _ = uuid.Parse(*req.CategoryID)
	}
	b, err := h.svc.Create(c.Request.Context(), userID, in)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, b)
}

// List handles GET /api/v1/budgets?month
func (h *Handler) List(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}
	budgets, err := h.svc.List(c.Request.Context(), userID, c.Query("month"))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, budgets)
}

// update handles PATCH /api/v1/budgets/:id.
func (h *Handler) Update(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}
	id, ok := budgetID(c)
	if !ok{
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	if req.Amount == nil && req.AlertThreshold == nil {
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", "send at least one of: amount, alert_threshold")
		return
	}

	b, err := h.svc.Update(c.Request.Context(), userID, id, UpdateInput{
		Amount: req.Amount, Threshold: req.AlertThreshold,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, b)
}

// Delete handles DELETE /api/v1/budgets/:id
func (h *Handler) Delete(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	id, ok := budgetID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func currentUser(c *gin.Context) (uuid.UUID, bool) {
	id, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
	}
	return id, ok
}

func budgetID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "budget id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "budget_not_found", "budget not found")
	case errors.Is(err, ErrAlreadyExists):
		httpx.Error(c, http.StatusConflict, "budget_exists", err.Error())
	case errors.Is(err, ErrCategoryNotFound):
		httpx.Error(c, http.StatusUnprocessableEntity, "invalid_category", "category not found")
	case errors.Is(err, ErrCategoryNotExpense), errors.Is(err, ErrInvalidAmount),
		errors.Is(err, ErrInvalidThreshold), errors.Is(err, ErrInvalidMonth):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		log.Printf("budget error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}