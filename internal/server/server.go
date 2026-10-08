package server

import(
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/config"
	"spendWise/internal/auth"
	"spendWise/internal/db"
	"spendWise/internal/user"
	"spendWise/pkg/httpx"
)

// 'new' wires every dependency together and returns the HTTP handlers
func New(cfg *config.Config, pool *pgxpool.Pool) http.Handler{
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	httpx.Init()

	// dependencies
	queries := db.New(pool)
	userRepo := user.NewRepository(queries)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := auth.NewService(userRepo, tokens)
	authHandler := auth.NewHandler(authSvc)

	// routes
	r := gin.Default() //includes request logging and panic recovery
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	v1 := r.Group("/api/v1")
	authGroup := v1.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)

	// Protected routes: everything on this group requires a valid JWT.
	protected := v1.Group("", auth.Middleware(tokens))
	protected.GET("/users/me", userHandler.Me)

	return r
}