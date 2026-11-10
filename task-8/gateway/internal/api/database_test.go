//go:build integration

package api

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func PrepareDatabase(t *testing.T, envKey, expectedName string) string {
	t.Helper()

	switch expectedName {
	case "cashapp_ledger_test", "cashapp_api_test":
	default:
		t.Fatalf("database is not allowed for tests: %q", expectedName)
	}

	dsn := os.Getenv(envKey)
	if dsn == "" {
		t.Fatalf("%s must be set explicitly", envKey)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal("invalid test database URL")
	}

	if (u.Scheme != "postgres" && u.Scheme != "postgresql") ||
		u.Path != "/"+expectedName {
		t.Fatalf("%s must point to database %q", envKey, expectedName)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test helper database: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var actualName string
	err = db.QueryRowContext(ctx, "SELECT current_database()").Scan(&actualName)
	if err != nil {
		t.Fatalf("check test database name: %v", err)
	}

	if actualName != expectedName {
		t.Fatalf(
			"refusing to clear database %q; expected %q",
			actualName,
			expectedName,
		)
	}

	const clearQuery = `
		TRUNCATE TABLE public.expenses, public.budgets RESTART IDENTITY
	`

	if _, err := db.ExecContext(ctx, clearQuery); err != nil {
		t.Fatalf("clear test database: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if _, err := db.ExecContext(cleanupCtx, clearQuery); err != nil {
			t.Errorf("clean test database after test: %v", err)
		}
	})

	return dsn
}
