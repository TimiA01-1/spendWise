package auth

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"spendWise/pkg/httpx"
)

type Handler struct{
	svc *Service
}

func NewHandler(svc *Service) *Handler{
	return &Handler{svc: svc}
}

type registerRequest struct{
	Name string `json:"name" binding:"required,min=2,max=100"`
	Email string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register handles POST /api/v1/auth/register
func (h *Handler) Register(c *gin.Context){
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err !=nil{
		httpx.BindError(c, err)
		return
	}

	u, err := h.svc.Register(c.Request.Context(), RegisterInput{
		Name: req.Name,
		Email: req.Email,
		Password: req.Password,
	})
	switch{
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(c, http.StatusConflict, "email_taken", "an account with this email already exists")
	case errors.Is(err, ErrPasswordTooLong):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	case err != nil:
		log.Printf("register failed: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	default:
		httpx.OK(c, http.StatusCreated, u)
	}
}

// login handles POST /api/v1/auth/login

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}

	res, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case err != nil:
		log.Printf("login failed: %v", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	default:
		httpx.OK(c, http.StatusOK, res)
	}
}