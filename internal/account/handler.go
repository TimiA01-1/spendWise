package account

import(
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"spendWise/internal/auth/authctx"
	"spendWise/pkg/httpx"
)

type Handler struct{
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createRequest struct{
	Name string `json:"name" binding:"required,min=2,max=100"`
	Type string `json:"type" binding:"required,oneof=bank cash wallet"`
	Currency string `json:"currency" binding:"omitempty,len=3"`
	OpeningBalance int64 `json:"opening_balance" binding:"gte=0"`
}

// pointers and omitempty ensures a field that is empty stays that way and is skipped
type updateRequest struct {
	Name *string `json:"name" binding:"omitempty,min=2,max=100"`
	Type *string `json:"type" binding:"omitempty,oneof=bank cash wallet"`
}

// create handles POST /api/v1/accounts
func (h*Handler) Create(c *gin.Context){
	userID, ok := currentUser(c)
	//userID, ok := authctx.UserID(c)
	if !ok{
		//httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil{
		httpx.BindError(c, err)
		return
	}

	acc, err := h.svc.Create(c.Request.Context(), userID, CreateInput(req))
	if err != nil{
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, acc)
	//switch{
	//case errors.Is(err, ErrNameTaken):
	//	httpx.Error(c, http.StatusConflict, "account_name_taken", err.Error())
	//case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidType), errors.Is(err, ErrNegativeBalance):
	//	httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	//case err != nil:
	//	log.Printf("create account failed: %v", err)
	//	httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	//default:
	//	httpx.OK(c, http.StatusCreated, acc)
	//}
}

// List handles GET /api/v1/accounts.
func (h *Handler) List(c *gin.Context){
	userID, ok := currentUser(c)
	//userID, ok := authctx.UserID(c)
	if !ok{
		//httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	accounts, err := h.svc.List(c.Request.Context(), userID)
	if err != nil{
		writeError(c, err)
		//log.Printf("list account failed; %v", err)
		//httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.OK(c, http.StatusOK, accounts)
}

// get handles GET /api/v1/accounts/:id
func (h *Handler) Get(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}
	id, ok := accountID(c)
	if !ok{
		return
	}
	acc, err := h.svc.Get(c.Request.Context(), userID, id)
	if err != nil{
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, acc)
}

// update handles PATCH /api/v1/accounts/:id
func (h *Handler) Update(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}
	id, ok := accountID(c)
	if !ok{
		return
	}
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	if req.Name == nil && req.Type == nil {
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", "send at least one of: name, type")
		return
	}
	acc, err := h.svc.Update(c.Request.Context(), userID, id, UpdateInput(req))
	if err != nil{
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, acc)
}

// delete handles DELETE /api/v1/accounts/:id
func (h *Handler) Delete(c *gin.Context){
	userID, ok := currentUser(c)
	if !ok{
		return
	}
	id, ok := accountID(c)
	if !ok{
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// current user reads authenticated user's Id, writing a 401 if its missing
func currentUser(c *gin.Context) (uuid.UUID, bool) {
	id, ok := authctx.UserID(c)
	if !ok{
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
	}
	return id, ok
}

// account parses the :id param, writing a 400 if it isnt a uuid
func accountID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil{
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "account id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

// writeError maps service errors to http responses in one place
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "account_not_found", "account not found")
	case errors.Is(err, ErrNameTaken):
		httpx.Error(c, http.StatusConflict, "account_name_taken", err.Error())
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidType), errors.Is(err, ErrNegativeBalance):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		log.Printf("account error: %v", err) // log the real error, never show it to the client
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}
