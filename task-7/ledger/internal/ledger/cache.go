package ledger

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var reportCache *redis.Client

func InitCache(ctx context.Context) error {
	dbNumber, err := strconv.Atoi(envOrDefault("REDIS_DB", "0"))
	if err != nil || dbNumber < 0 {
		return fmt.Errorf("REDIS_DB must be a non-negative integer")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         envOrDefault("REDIS_ADDR", "localhost:6379"),
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           dbNumber,
		Protocol:     2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("ping Redis: %w", err)
	}

	reportCache = client
	return nil
}

func CloseCache() error {
	if reportCache == nil {
		return nil
	}

	return reportCache.Close()
}
