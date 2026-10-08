package auth

import(
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"spendWise/internal/auth/authctx"
)

func TestMiddleware(t *testing.T){
	gin.SetMode(gin.TestMode)

	tokens := NewTokenManager(testSecret, time.Hour)
	userID := uuid.New()
	valid, _, _ := tokens.Generate(userID)

	r := gin.New()
	r.GET("/protected", Middleware(tokens), func(c *gin.Context) {
		id, ok := authctx.UserID(c)
		if !ok || id != userID {
			t.Errorf("authctx.UserID = %v, %v; want %v, true", id, ok, userID)
		}
		c.Status(http.StatusOK)
	})

	tests := []struct{
		name string
		header string
		wantStatus int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic abc123", http.StatusUnauthorized},
		{"bearer without a token", "Bearer ", http.StatusUnauthorized},
		{"invalid token", "Bearer nope", http.StatusUnauthorized},
		{"valid token", "Bearer " + valid, http.StatusOK},
		{"scheme is case-insensitive", "bearer " + valid, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}