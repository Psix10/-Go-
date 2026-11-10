package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"ledger/internal/domain"
)

const (
	reportCacheTTL     = 30 * time.Second
	reportCacheTimeout = time.Second
)

type ReportService struct {
	repository domain.ReportRepository
	cache      domain.ReportCache
}

func NewReportService(
	repository domain.ReportRepository,
	cache domain.ReportCache,
) *ReportService {
	return &ReportService{
		repository: repository,
		cache:      cache,
	}
}

func (s *ReportService) GetReportSummary(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (domain.ReportSummary, error) {
	fromDate := from.Format("2006-01-02")
	toDate := to.Format("2006-01-02")

	if fromDate > toDate {
		return domain.ReportSummary{}, fmt.Errorf(
			"from must not be after to",
		)
	}

	key := "report:summary:" + fromDate + ":" + toDate

	if s.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, reportCacheTimeout)
		report, found, err := s.cache.Get(cacheCtx, key)
		cancel()

		switch {
		case err != nil:
			log.Printf("report cache read failed: %v", err)

		case found:
			log.Printf("report cache HIT: %s", key)
			return report, nil

		default:
			log.Printf("report cache MISS: %s", key)
		}
	}

	categories, err := s.repository.SumByCategoryBetween(ctx, from, to)
	if err != nil {
		return domain.ReportSummary{}, err
	}

	if categories == nil {
		categories = make([]domain.CategorySummary, 0)
	}

	report := domain.ReportSummary{
		From:       fromDate,
		To:         toDate,
		Categories: categories,
	}

	for _, category := range categories {
		report.Total += category.Amount
	}

	if s.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, reportCacheTimeout)
		err := s.cache.Set(cacheCtx, key, report, reportCacheTTL)
		cancel()

		if err != nil {
			log.Printf("report cache write failed: %v", err)
		} else {
			log.Printf("report cache SET: %s, TTL=30s", key)
		}
	}

	return report, nil
}
