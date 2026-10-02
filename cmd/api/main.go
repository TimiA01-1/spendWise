package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"spendWise/config"
	//"spendWise/internal/db"
	"spendWise/internal/server"
	//"spendWise/internal/user"
	"spendWise/pkg/postgres"
)

func main(){
	if err := run(); err != nil{
		log.Fatalf("fatal: %v", err)
	}
}
func run() error{
	cfg, err := config.Load()
	if err != nil{
		return fmt.Errorf("load config: %v", err)
	}
	// ctx cancels when ctrl c 
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	//ctx := context.Background()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil{
		return fmt.Errorf("connect database: %v", err)
	}
	defer pool.Close()

	srv := &http.Server{
		Addr: ":" + cfg.HTTPPort,
		Handler: server.New(cfg, pool),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	// in flight request have up to 10 seconds to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Println("server stopped cleanly")
	return nil

	//repo := user.NewRepository(db.New(pool))
	//temp
	//u, err := repo.Create(ctx, "Ada Lovelace", "ada@example.com", "fake-hash")
	//if errors.Is(err, user.ErrEmailTaken) {
	//	log.Println("create: email already taken (as expected on 2nd run)")
	//} else if err != nil {
	//	log.Fatalf("create: %v", err)
	//} else {
	//	log.Printf("created: %+v", u)
	//}

	//found, err := repo.GetByEmail(ctx, "ADA@Example.com")
	//log.Printf("lookup (mixed case): id=%s name=%s err=%v", found.ID, found.Name, err)

	//_, err = repo.GetByEmail(ctx, "ghost@example.com")
	//log.Printf("lookup unknown: is ErrNotFound? %v", errors.Is(err, user.ErrNotFound))

	//queries := db.New(pool)

	//var userCount int
	//if err := pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&userCount); err != nil {
	//	log.Fatalf("query users: %v", err)
	//}
	//log.Printf("database connected, users in table: %d", userCount)

	//u, err := queries.GetUserByEmail(ctx, "nobody@example.com")
	//log.Printf("user: %+v, err: %v", u, err)

	//fmt.Printf("SpendWise API starting on port %s (env: %s)\n", cfg.HTTPPort, cfg.Env)
}