package server

import(
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/config"
	"spendWise/internal/account"
	"spendWise/internal/auth"
	"spendWise/internal/db"
	"spendWise/internal/user"
	"spendWise/internal/transaction"
	"spendWise/internal/budget"
	"spendWise/internal/category"
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
	accountRepo := account.NewRepository(queries)
	accountSvc := account.NewService(accountRepo)
	accountHandler := account.NewHandler(accountSvc)
	categoryRepo := category.NewRepository(queries)
	categorySvc := category.NewService(categoryRepo)
	categoryHandler := category.NewHandler(categorySvc)
	txRepo := transaction.NewRepository(pool, queries)
	txSvc := transaction.NewService(txRepo)
	txHandler := transaction.NewHandler(txSvc)
	budgetRepo := budget.NewRepository(queries)
	budgetSvc := budget.NewService(budgetRepo)
	budgetHandler := budget.NewHandler(budgetSvc)
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
	protected.POST("/accounts", accountHandler.Create)
	protected.GET("/accounts", accountHandler.List)
	protected.GET("/accounts/:id", accountHandler.Get)
	protected.PATCH("/accounts/:id", accountHandler.Update)
	protected.DELETE("/accounts/:id", accountHandler.Delete)
	protected.GET("/categories", categoryHandler.List)
	protected.POST("/categories", categoryHandler.Create)
	protected.POST("/transactions", txHandler.Create)
	protected.GET("/transactions", txHandler.List)
	protected.GET("/transactions/:id", txHandler.Get)
	protected.PATCH("/transactions/:id", txHandler.Update)
	protected.DELETE("/transactions/:id", txHandler.Delete)
	protected.POST("/budgets", budgetHandler.Create)
	protected.GET("/budgets", budgetHandler.List)
	protected.PATCH("/budgets/:id", budgetHandler.Update)
	protected.DELETE("/budgets/:id", budgetHandler.Delete)

	return r
}