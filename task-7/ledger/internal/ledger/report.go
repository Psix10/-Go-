package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type CategorySummary struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

type ReportSummary struct {
	From       string            `json:"from"`
	To         string            `json:"to"`
	Total      float64           `json:"total"`
	Categories []CategorySummary `json:"categories"`
}

func GetReportSummary(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (ReportSummary, error) {
	fromDate := from.Format("2006-01-02")
	toDate := to.Format("2006-01-02")

	if fromDate > toDate {
		return ReportSummary{}, fmt.Errorf("from must not be after to")
	}

	key := "report:summary:" + fromDate + ":" + toDate

	if reportCache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, time.Second)
		data, err := reportCache.Get(cacheCtx, key).Bytes()
		cancel()

		switch {
		case err == nil:
			var cached ReportSummary

			if err := json.Unmarshal(data, &cached); err == nil {
				log.Printf("report cache HIT: %s", key)
				return cached, nil
			} else {
				log.Printf("report cache invalid JSON: %v", err)
			}

		case errors.Is(err, redis.Nil):
			log.Printf("report cache MISS: %s", key)

		default:
			log.Printf("report cache read failed: %v", err)
		}
	}

	if database == nil {
		return ReportSummary{}, fmt.Errorf("PostgreSQL is not initialized")
	}

	report := ReportSummary{
		From:       fromDate,
		To:         toDate,
		Categories: make([]CategorySummary, 0),
	}

	const query = `
		SELECT category, SUM(amount)
		FROM expenses
		WHERE date >= $1::date AND date <= $2::date
		GROUP BY category
		ORDER BY category
	`

	err := func() error {
		rows, err := database.QueryContext(ctx, query, fromDate, toDate)
		if err != nil {
			return fmt.Errorf("query report summary: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item CategorySummary

			if err := rows.Scan(&item.Category, &item.Amount); err != nil {
				return fmt.Errorf("scan report category: %w", err)
			}

			report.Categories = append(report.Categories, item)
			report.Total += item.Amount
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate report categories: %w", err)
		}

		return nil
	}()
	if err != nil {
		return ReportSummary{}, err
	}

	if reportCache != nil {
		data, err := json.Marshal(report)
		if err != nil {
			return ReportSummary{}, fmt.Errorf("marshal report: %w", err)
		}

		cacheCtx, cancel := context.WithTimeout(ctx, time.Second)
		err = reportCache.Set(cacheCtx, key, data, 30*time.Second).Err()
		cancel()

		if err != nil {
			log.Printf("report cache write failed: %v", err)
		} else {
			log.Printf("report cache SET: %s, TTL=30s", key)
		}
	}

	return report, nil
}
