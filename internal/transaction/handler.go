package transaction

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

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
	AccountID  string `json:"account_id" binding:"required,uuid"`
	CategoryID string `json:"category_id" binding:"required,uuid"`
	Type  string `json:"type" binding:"required,oneof=income expense"`
	Amount int64  `json:"amount" binding:"required,gt=0"`
	Note string `json:"note" binding:"max=500"`
	Date string `json:"date" binding:"omitempty,datetime=2006-01-02"`
}

// create handles POST /api/v1/transactions
func (h *Handler) Create(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		//httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}

	// the validator already proved these strings are well-formed
	accountID, _ := uuid.Parse(req.AccountID)
	categoryID, _ := uuid.Parse(req.CategoryID)
	var date *time.Time
	if req.Date != "" {
		d, _ := time.Parse(dateLayout, req.Date)
		date = &d
	}

	t, err := h.svc.Create(c.Request.Context(), userID, CreateInput{
		AccountID: accountID,
		CategoryID: categoryID,
		Type: req.Type,
		Amount: req.Amount,
		Note: req.Note,
		Date: date,
	})
	if err != nil{
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, t)
}

type updateRequest struct {
	AccountID  *string `json:"account_id" binding:"omitempty,uuid"`
	CategoryID *string `json:"category_id" binding:"omitempty,uuid"`
	Type       *string `json:"type" binding:"omitempty,oneof=income expense"`
	Amount     *int64  `json:"amount" binding:"omitempty,gt=0"`
	Note       *string `json:"note" binding:"omitempty,max=500"`
	Date       *string `json:"date" binding:"omitempty,datetime=2006-01-02"`
}

// Get handles GET /api/v1/transactions/:id.
func (h *Handler) Get(c *gin.Context) {
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	id, ok := transactionID(c)
	if !ok {
		return
	}

	t, err := h.svc.Get(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, t)
}

// update handles PATCH /api/v1/transactions/:id
func (h *Handler) Update(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}
	id, ok := transactionID(c)
	if !ok{
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil{
		httpx.BindError(c, err)
		return
	}
	if req == (updateRequest{}) {
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed",
			"send at least one of: account_id, category_id, type, amount, note, date")
		return
	}

	in := UpdateInput{Type: req.Type, Amount: req.Amount, Note: req.Note}
	if req.AccountID != nil {
		v, _ := uuid.Parse(*req.AccountID)
		in.AccountID = &v
	}
	if req.CategoryID != nil {
		v, _ := uuid.Parse(*req.CategoryID)
		in.CategoryID = &v
	}
	if req.Date != nil {
		d, _ := time.Parse(dateLayout, *req.Date)
		in.Date = &d
	}

	t, err := h.svc.Update(c.Request.Context(), userID, id, in)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, t)
}

// delete handles DELETE /api/v1/transactions/:id.
func (h *Handler) Delete(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	id, ok := transactionID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// currentUser reads the authenticated user's ID, writing a 401 if it's missing.
func currentUser(c *gin.Context) (uuid.UUID, bool) {
	id, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
	}
	return id, ok
}

// transactionID parses the :id path parameter, writing a 400 if it isn't a UUID.
func transactionID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "transaction id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

// list handles GET /api/v1/transactions
func (h *Handler) List(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok {
		//httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	in := ListInput{Type: c.Query("type")}
	var err error

	if in.AccountID, err = optionalUUID(c, "account_id"); err != nil {
		badQuery(c, "account_id must be a valid UUID")
		return
	}
	if in.CategoryID, err = optionalUUID(c, "category_id"); err != nil {
		badQuery(c, "category_id must be a valid UUID")
		return
	}
	if in.From, err = optionalDate(c, "from"); err != nil {
		badQuery(c, "from must be a date like 2026-10-31")
		return
	}
	if in.To, err = optionalDate(c, "to"); err != nil {
		badQuery(c, "to must be a date like 2026-10-31")
		return
	}
	if in.Limit, err = optionalInt(c, "limit"); err != nil {
		badQuery(c, "limit must be a whole number")
		return
	}
	if in.Offset, err = optionalInt(c, "offset"); err != nil {
		badQuery(c, "offset must be a whole number")
		return
	}

	txs, err := h.svc.List(c.Request.Context(), userID, in)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, txs)
}

func optionalUUID(c *gin.Context, key string)(uuid.UUID, error){
	v := c.Query(key)
	if v == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(v)
}

func optionalDate(c *gin.Context, key string)(*time.Time, error){
	v := c.Query(key)
	if v == "" {
		return nil, nil
	}
	d, err := time.Parse(dateLayout, v)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func optionalInt(c *gin.Context, key string)(int, error){
	v := c.Query(key)
	if v == "" {
		return 0, nil
	}
	return strconv.Atoi(v)
}

func badQuery(c *gin.Context, msg string) {
	httpx.Error(c, http.StatusBadRequest, "invalid_query", msg)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "transaction_not_found", "transaction not found")
	case errors.Is(err, ErrAccountNotFound):
		httpx.Error(c, http.StatusUnprocessableEntity, "invalid_account", "account not found")
	case errors.Is(err, ErrCategoryNotFound):
		httpx.Error(c, http.StatusUnprocessableEntity, "invalid_category", "category not found")
	case errors.Is(err, ErrCategoryTypeMismatch), errors.Is(err, ErrInvalidType),
		errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrNoteTooLong), errors.Is(err, ErrInvalidRange):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		log.Printf("transaction error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}