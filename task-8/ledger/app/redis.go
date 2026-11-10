package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func openRedis(ctx context.Context) (*redis.Client, error) {
	dbNumber, err := strconv.Atoi(envOrDefault("REDIS_DB", "0"))
	if err != nil || dbNumber < 0 {
		return nil, fmt.Errorf("REDIS_DB must be a non-negative integer")
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
		return nil, fmt.Errorf("ping Redis: %w", err)
	}

	return client, nil
}
