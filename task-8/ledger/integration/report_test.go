//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestReportSummaryCache(t *testing.T) {
	svc := newTestService(t)

	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       15,
		Protocol: 2,
	})

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close test Redis inspector: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping test Redis inspector: %v", err)
	}
	from := time.Date(2040, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2040, time.January, 31, 0, 0, 0, 0, time.UTC)

	const key = "report:summary:2040-01-01:2040-01-31"

	if err := client.Del(ctx, key).Err(); err != nil {
		t.Fatalf("delete previous test key: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := client.Del(cleanupCtx, key).Err(); err != nil {
			t.Errorf("delete test report key: %v", err)
		}
	})

	addExpense := func(amount float64, category string, date time.Time) {
		t.Helper()

		_, err := svc.AddTransaction(ctx, Transaction{
			Amount:      amount,
			Category:    category,
			Description: "report test",
			Date:        date,
		})
		if err != nil {
			t.Fatalf("add report test expense: %v", err)
		}
	}

	// Обе граничные даты должны входить в отчёт.
	addExpense(100, "food", from)
	addExpense(200, "transport", to)

	// Эта запись за пределами периода и не должна попасть в отчёт.
	addExpense(900, "outside", to.AddDate(0, 0, 1))

	first, err := svc.GetReportSummary(ctx, from, to)
	if err != nil {
		t.Fatalf("calculate first report: %v", err)
	}

	if first.From != "2040-01-01" || first.To != "2040-01-31" {
		t.Fatalf("unexpected report interval: %+v", first)
	}

	if first.Total != 300 {
		t.Fatalf("expected total=300, got %v", first.Total)
	}

	if len(first.Categories) != 2 {
		t.Fatalf("expected two categories, got %+v", first.Categories)
	}

	if first.Categories[0].Category != "food" ||
		first.Categories[0].Amount != 100 ||
		first.Categories[1].Category != "transport" ||
		first.Categories[1].Amount != 200 {
		t.Fatalf("unexpected categories: %+v", first.Categories)
	}

	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("read report TTL: %v", err)
	}

	if ttl <= 0 || ttl > 30*time.Second {
		t.Fatalf("expected TTL in (0, 30s], got %v", ttl)
	}

	// Меняем PostgreSQL после заполнения кэша.
	addExpense(50, "food", from)

	cached, err := svc.GetReportSummary(ctx, from, to)
	if err != nil {
		t.Fatalf("read cached report: %v", err)
	}

	// Если сработал HIT, сумма пока останется прежней.
	if cached.Total != 300 {
		t.Fatalf("expected cached total=300, got %v", cached.Total)
	}

	// Удаляем только свой ключ, чтобы проверить пересчёт без ожидания TTL.
	if err := client.Del(ctx, key).Err(); err != nil {
		t.Fatalf("invalidate test report: %v", err)
	}

	refreshed, err := svc.GetReportSummary(ctx, from, to)
	if err != nil {
		t.Fatalf("refresh report: %v", err)
	}

	if refreshed.Total != 350 {
		t.Fatalf("expected refreshed total=350, got %v", refreshed.Total)
	}

	if len(refreshed.Categories) != 2 ||
		refreshed.Categories[0].Category != "food" ||
		refreshed.Categories[0].Amount != 150 {
		t.Fatalf("unexpected refreshed categories: %+v", refreshed.Categories)
	}
}
