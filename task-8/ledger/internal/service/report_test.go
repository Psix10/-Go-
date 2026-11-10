package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ledger/internal/domain"
)

type reportRepositoryStub struct {
	categories []domain.CategorySummary
	err        error
	calls      int
}

func (r *reportRepositoryStub) SumByCategoryBetween(
	_ context.Context,
	_ time.Time,
	_ time.Time,
) ([]domain.CategorySummary, error) {
	r.calls++
	return r.categories, r.err
}

type reportCacheStub struct {
	report   domain.ReportSummary
	found    bool
	getErr   error
	setErr   error
	getCalls int
	setCalls int
	key      string
	ttl      time.Duration
}

func (c *reportCacheStub) Get(
	_ context.Context,
	key string,
) (domain.ReportSummary, bool, error) {
	c.getCalls++
	c.key = key
	return c.report, c.found, c.getErr
}

func (c *reportCacheStub) Set(
	_ context.Context,
	key string,
	report domain.ReportSummary,
	ttl time.Duration,
) error {
	c.setCalls++
	c.key = key
	c.report = report
	c.ttl = ttl
	return c.setErr
}

func TestGetReportSummary(t *testing.T) {
	cacheErr := errors.New("cache unavailable")

	tests := []struct {
		name          string
		found         bool
		getErr        error
		setErr        error
		wantRepoCalls int
		wantSetCalls  int
		wantTotal     float64
	}{
		{
			name:          "cache miss",
			wantRepoCalls: 1,
			wantSetCalls:  1,
			wantTotal:     50,
		},
		{
			name:      "cache hit",
			found:     true,
			wantTotal: 99,
		},
		{
			name:          "cache read error",
			getErr:        cacheErr,
			wantRepoCalls: 1,
			wantSetCalls:  1,
			wantTotal:     50,
		},
		{
			name:          "cache write error",
			setErr:        cacheErr,
			wantRepoCalls: 1,
			wantSetCalls:  1,
			wantTotal:     50,
		},
	}

	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &reportRepositoryStub{
				categories: []domain.CategorySummary{
					{Category: "food", Amount: 30},
					{Category: "transport", Amount: 20},
				},
			}

			cache := &reportCacheStub{
				report: domain.ReportSummary{
					From:  "2026-11-01",
					To:    "2026-11-10",
					Total: 99,
				},
				found:  tt.found,
				getErr: tt.getErr,
				setErr: tt.setErr,
			}

			svc := NewReportService(repository, cache)

			report, err := svc.GetReportSummary(
				context.Background(),
				from,
				to,
			)
			if err != nil {
				t.Fatal(err)
			}

			if report.Total != tt.wantTotal {
				t.Errorf("total = %v, want %v", report.Total, tt.wantTotal)
			}

			if report.From != "2026-11-01" || report.To != "2026-11-10" {
				t.Errorf("unexpected report dates: %+v", report)
			}

			if repository.calls != tt.wantRepoCalls {
				t.Errorf(
					"repository calls = %d, want %d",
					repository.calls,
					tt.wantRepoCalls,
				)
			}

			if cache.getCalls != 1 {
				t.Errorf("cache get calls = %d, want 1", cache.getCalls)
			}

			if cache.setCalls != tt.wantSetCalls {
				t.Errorf(
					"cache set calls = %d, want %d",
					cache.setCalls,
					tt.wantSetCalls,
				)
			}

			if cache.key != "report:summary:2026-11-01:2026-11-10" {
				t.Errorf("unexpected cache key: %q", cache.key)
			}

			if cache.setCalls > 0 && cache.ttl != 30*time.Second {
				t.Errorf("TTL = %v, want 30s", cache.ttl)
			}
		})
	}
}

func TestReportInvalidPeriod(t *testing.T) {
	repository := &reportRepositoryStub{}
	cache := &reportCacheStub{}
	svc := NewReportService(repository, cache)

	from := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, -1)

	_, err := svc.GetReportSummary(context.Background(), from, to)
	if err == nil {
		t.Fatal("expected invalid period error")
	}

	if repository.calls != 0 || cache.getCalls != 0 || cache.setCalls != 0 {
		t.Fatal("invalid period must not access storage")
	}
}

func TestReportWithoutCache(t *testing.T) {
	repository := &reportRepositoryStub{}
	svc := NewReportService(repository, nil)

	date := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	report, err := svc.GetReportSummary(context.Background(), date, date)
	if err != nil {
		t.Fatal(err)
	}

	if repository.calls != 1 {
		t.Errorf("repository calls = %d, want 1", repository.calls)
	}

	if report.Total != 0 {
		t.Errorf("total = %v, want 0", report.Total)
	}

	if report.Categories == nil || len(report.Categories) != 0 {
		t.Errorf("expected non-nil empty categories, got %#v", report.Categories)
	}
}

func TestReportRepositoryError(t *testing.T) {
	storageErr := errors.New("database unavailable")
	repository := &reportRepositoryStub{err: storageErr}
	cache := &reportCacheStub{}
	svc := NewReportService(repository, cache)

	date := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	_, err := svc.GetReportSummary(context.Background(), date, date)
	if !errors.Is(err, storageErr) {
		t.Fatalf("error = %v, want %v", err, storageErr)
	}

	if cache.setCalls != 0 {
		t.Fatal("failed report must not be cached")
	}
}
