package category

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"spendWise/internal/auth/authctx"
	"spendWise/pkg/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createRequest struct{
	Name string `json:"name" binding:"required,min=2,max=50"`
	Type string `json:"type" binding:"required,oneof=income expense"`
}

// list handles GET /api/v1/categories
func (h *Handler) List(c *gin.Context) {
	userID, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	cats, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, cats)
}

// create handles POST /api/v1/categories
func (h *Handler) Create(c *gin.Context) {
	userID, ok := authctx.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}

	cat, err := h.svc.Create(c.Request.Context(), userID, req.Name, req.Type)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, cat)
}

func writeError(c *gin.Context, err error){
	switch {
	case errors.Is(err, ErrNameTaken):
		httpx.Error(c, http.StatusConflict, "category_name_taken", err.Error())
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidType):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		log.Printf("category error: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}