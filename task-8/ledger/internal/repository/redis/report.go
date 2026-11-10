package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ledger/internal/domain"

	goredis "github.com/redis/go-redis/v9"
)

type ReportCache struct {
	client *goredis.Client
}

func NewReportCache(client *goredis.Client) *ReportCache {
	return &ReportCache{client: client}
}

var _ domain.ReportCache = (*ReportCache)(nil)

func (c *ReportCache) Get(
	ctx context.Context,
	key string,
) (domain.ReportSummary, bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()

	if errors.Is(err, goredis.Nil) {
		return domain.ReportSummary{}, false, nil
	}

	if err != nil {
		return domain.ReportSummary{}, false, fmt.Errorf(
			"read report cache: %w",
			err,
		)
	}

	var report domain.ReportSummary

	if err := json.Unmarshal(data, &report); err != nil {
		return domain.ReportSummary{}, false, fmt.Errorf(
			"decode report cache: %w",
			err,
		)
	}

	return report, true, nil
}

func (c *ReportCache) Set(
	ctx context.Context,
	key string,
	report domain.ReportSummary,
	ttl time.Duration,
) error {
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("write report cache: %w", err)
	}

	return nil
}
