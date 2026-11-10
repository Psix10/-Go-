package domain

import (
	"context"
	"time"
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

type ReportRepository interface {
	SumByCategoryBetween(
		ctx context.Context,
		from time.Time,
		to time.Time,
	) ([]CategorySummary, error)
}

type ReportCache interface {
	Get(
		ctx context.Context,
		key string,
	) (ReportSummary, bool, error)

	Set(
		ctx context.Context,
		key string,
		report ReportSummary,
		ttl time.Duration,
	) error
}
