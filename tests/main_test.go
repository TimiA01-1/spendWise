// pacakage tests holds the integration tests
package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"spendWise/config"
	"spendWise/internal/server"
)

const testJWTSecret = "integration-test-secret-at-least-32-characters"

var (
	testPool *pgxpool.Pool
	testHandler http.Handler
)

func TestMain(m *testing.M) { os.Exit(run(m)) }

func run(m *testing.M) int{
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		fmt.Println("integration tests skipped: set TEST_DATABASE_URL (see the comment in tests/main_test.go)")
		return 0
	}
	ctx := context.Background()

	// a brand new database for this run, so tests never see leftovers
	dbName := "spendwise_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if err := execAdmin(ctx, adminURL, "CREATE DATABASE "+dbName); err != nil {
		fmt.Fprintln(os.Stderr, "create test database:", err)
		return 1
	}
	defer func() {
		if err := execAdmin(ctx, adminURL, "DROP DATABASE "+dbName+" WITH (FORCE)"); err != nil {
			fmt.Fprintln(os.Stderr, "drop test database:", err)
		}
	}()

	cfg, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse TEST_DATABASE_URL:", err)
		return 1
	}
	cfg.ConnConfig.Database = dbName
	cfg.MaxConns = 20 // concurrency tests fire many requests at once
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect to test database:", err)
		return 1
	}
	defer pool.Close() // runs before the drop above (deferred later = runs first)

	if err := applyMigrations(ctx, pool, "../migrations"); err != nil {
		fmt.Fprintln(os.Stderr, "apply migrations:", err)
		return 1
	}

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard // silence the per-request access log
	testPool = pool
	testHandler = server.New(&config.Config{
		Env:               "test",
		JWTSecret:         testJWTSecret,
		JWTTTL:            time.Hour,
		RecurringInterval: time.Minute,
	}, pool)

	return m.Run()
}

func execAdmin(ctx context.Context, url, sql string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, sql)
	return err
}

// applyMigration runs every *.up.sql file in order
func applyMigrations(ctx context.Context, pool *pgxpool.Pool, dir string)error{
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no migrations found in %s", dir)
	}
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}
