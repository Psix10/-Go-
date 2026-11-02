package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var database *sql.DB

func InitDB(ctx context.Context) error {
	db, err := sql.Open("pgx", databaseDSN())
	if err != nil {
		return fmt.Errorf("open PostgreSQL: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}

	database = db
	return nil
}

func CloseDB() error {
	if database == nil {
		return nil
	}
	return database.Close()
}

func databaseDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	u := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			envOrDefault("DB_USER", "postgres"),
			envOrDefault("DB_PASS", "postgres"),
		),
		Host: net.JoinHostPort(
			envOrDefault("DB_HOST", "localhost"),
			envOrDefault("DB_PASS", "postgres"),
		),
		Path: "/" + envOrDefault("DB_NAME", "cashapp"),
	}

	query := u.Query()
	query.Set("sslmode", "disable")
	u.RawQuery = query.Encode()

	return u.String()
}

func envOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
