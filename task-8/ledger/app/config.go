package app

import (
	"net"
	"net/url"
	"os"
)

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
			envOrDefault("DB_PORT", "5432"),
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
