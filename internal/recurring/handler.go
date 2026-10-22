package recurring

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"spendWise/internal/auth/authctx"
	"spendWise/internal/transaction"
	"spendWise/pkg/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createRequest struct {
	AccountID  string `json:"account_id" binding:"required,uuid"`
	CategoryID string `json:"category_id" binding:"required,uuid"`
	Type string `json:"type" binding:"required,oneof=income expense"`
	Amount int64  `json:"amount" binding:"required,gt=0"`
	Note string `json:"note" binding:"max=500"`
	Frequency  string `json:"frequency" binding:"required,oneof=daily weekly monthly yearly"`
	StartDate  string `json:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate string `json:"end_date" binding:"omitempty,datetime=2006-01-02"`
}

// Pointers + omitempty: a field that is absent stays nil and is left unchanged.
type updateRequest struct {
	Amount  *int64  `json:"amount" binding:"omitempty,gt=0"`
	Note *string `json:"note" binding:"omitempty,max=500"`
	EndDate *string `json:"end_date" binding:"omitempty,datetime=2006-01-02"`
	Status  *string `json:"status" binding:"omitempty,oneof=active paused"`
}

// Create handles POST /api/v1/recurring-transactions
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

	// The validator already proved these strings are well-formed.
	accountID, _ := uuid.Parse(req.AccountID)
	categoryID, _ := uuid.Parse(req.CategoryID)
	start, _ := time.Parse(dateLayout, req.StartDate)
	var end *time.Time
	if req.EndDate != "" {
		e, _ := time.Parse(dateLayout, req.EndDate)
		end = &e
	}

	rule, err := h.svc.Create(c.Request.Context(), userID, CreateInput{
		AccountID:  accountID,
		CategoryID: categoryID,
		Type: req.Type,
		Amount: req.Amount,
		Note: req.Note,
		Frequency:  req.Frequency,
		StartDate:  start,
		EndDate: end,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, rule.View())
}

// List handles GET /api/v1/recurring-transactions
func (h *Handler) List(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok {
		return
	}

	rules, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}
	views := make([]RuleView, 0, len(rules))
	for _, r := range rules {
		views = append(views, r.View())
	}
	httpx.OK(c, http.StatusOK, views)
}

// Update handles PATCH /api/v1/recurring-transactions/:id
func (h *Handler) Update(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	id, ok := ruleID(c)
	if !ok {
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	if req == (updateRequest{}) {
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed",
			"send at least one of: amount, note, end_date, status")
		return
	}

	in := UpdateInput{Amount: req.Amount, Note: req.Note, Status: req.Status}
	if req.EndDate != nil {
		e, _ := time.Parse(dateLayout, *req.EndDate)
		in.EndDate = &e
	}

	rule, err := h.svc.Update(c.Request.Context(), userID, id, in)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rule.View())
}

// delete handles DELETE /api/v1/recurring-transactions/:id
func (h *Handler) Delete(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	id, ok := ruleID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		writeError(c, err)
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

func ruleID(c *gin.Context)(uuid.UUID, bool){
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "recurring transaction id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "recurring_not_found", "recurring transaction not found")
	case errors.Is(err, ErrFinished):
		httpx.Error(c, http.StatusConflict, "rule_finished", err.Error())
	case errors.Is(err, transaction.ErrAccountNotFound):
		httpx.Error(c, http.StatusUnprocessableEntity, "invalid_account", "account not found")
	case errors.Is(err, transaction.ErrCategoryNotFound):
		httpx.Error(c, http.StatusUnprocessableEntity, "invalid_category", "category not found")
	case errors.Is(err, transaction.ErrCategoryTypeMismatch), errors.Is(err, transaction.ErrInvalidType),
		errors.Is(err, transaction.ErrInvalidAmount), errors.Is(err, transaction.ErrNoteTooLong),
		errors.Is(err, ErrInvalidFrequency), errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrStartTooOld), errors.Is(err, ErrEndBeforeStart):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		log.Printf("recurring error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}
