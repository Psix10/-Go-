package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"ledger/service"
)

type serviceStub struct {
	err    error
	called string
	tx     service.Transaction
	budget service.Budget
	from   time.Time
	to     time.Time
}

var _ service.Service = (*serviceStub)(nil)

func (s *serviceStub) SetBudget(
	_ context.Context,
	budget service.Budget,
) error {
	s.called = "SetBudget"
	s.budget = budget
	return s.err
}

func (s *serviceStub) ListBudgets(
	_ context.Context,
) ([]service.Budget, error) {
	s.called = "ListBudgets"

	return []service.Budget{
		{Category: "food", Limit: 100, Period: "month"},
	}, s.err
}

func (s *serviceStub) AddTransaction(
	_ context.Context,
	tx service.Transaction,
) (service.Transaction, error) {
	s.called = "AddTransaction"
	s.tx = tx
	tx.ID = 42
	return tx, s.err
}

func (s *serviceStub) ListTransactions(
	_ context.Context,
) ([]service.Transaction, error) {
	s.called = "ListTransactions"
	return []service.Transaction{}, s.err
}

func (s *serviceStub) GetReportSummary(
	_ context.Context,
	from time.Time,
	to time.Time,
) (service.ReportSummary, error) {
	s.called = "GetReportSummary"
	s.from = from
	s.to = to

	return service.ReportSummary{
		From:       from.Format("2006-01-02"),
		To:         to.Format("2006-01-02"),
		Total:      0,
		Categories: []service.CategorySummary{},
	}, s.err
}

func TestGatewayHandlers(t *testing.T) {
	validationErr := fmt.Errorf(
		"invalid budget: %w",
		&service.ValidationError{
			Err: errors.New("budget limit must be a finite positive number"),
		},
	)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		serviceErr error
		wantStatus int
		wantBody   string
		wantCall   string
	}{
		{
			name:       "list budgets",
			method:     http.MethodGet,
			path:       "/api/budgets",
			wantStatus: http.StatusOK,
			wantBody:   `[{"category":"food","limit":100,"period":"month"}]`,
			wantCall:   "ListBudgets",
		},
		{
			name:       "create budget",
			method:     http.MethodPost,
			path:       "/api/budgets",
			body:       `{"category":"food","limit":100}`,
			wantStatus: http.StatusCreated,
			wantBody:   `{"category":"food","limit":100,"period":"month"}`,
			wantCall:   "SetBudget",
		},
		{
			name:       "list transactions",
			method:     http.MethodGet,
			path:       "/api/transactions",
			wantStatus: http.StatusOK,
			wantBody:   `[]`,
			wantCall:   "ListTransactions",
		},
		{
			name:       "create transaction",
			method:     http.MethodPost,
			path:       "/api/transactions",
			body:       `{"amount":10,"category":"food","description":"lunch","date":"2026-11-10"}`,
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":42,"amount":10,"category":"food","description":"lunch","date":"2026-11-10"}`,
			wantCall:   "AddTransaction",
		},
		{
			name:       "invalid JSON",
			method:     http.MethodPost,
			path:       "/api/budgets",
			body:       `{"category":`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid JSON"}`,
		},
		{
			name:       "missing transaction date",
			method:     http.MethodPost,
			path:       "/api/transactions",
			body:       `{"amount":10,"category":"food"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"transaction date cannot be empty"}`,
		},
		{
			name:       "invalid transaction date",
			method:     http.MethodPost,
			path:       "/api/transactions",
			body:       `{"amount":10,"category":"food","date":"wrong"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid date \"wrong\", expected format YYYY-MM-DD"}`,
		},
		{
			name:       "service validation",
			method:     http.MethodPost,
			path:       "/api/budgets",
			body:       `{"category":"food","limit":0}`,
			serviceErr: validationErr,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"budget limit must be a finite positive number"}`,
			wantCall:   "SetBudget",
		},
		{
			name:       "budget exceeded",
			method:     http.MethodPost,
			path:       "/api/transactions",
			body:       `{"amount":999,"category":"food","date":"2026-11-10"}`,
			serviceErr: fmt.Errorf("category food: %w", service.ErrBudgetExceeded),
			wantStatus: http.StatusConflict,
			wantBody:   `{"error":"budget exceeded"}`,
			wantCall:   "AddTransaction",
		},
		{
			name:       "storage error",
			method:     http.MethodGet,
			path:       "/api/budgets",
			serviceErr: errors.New("private database error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"internal server error"}`,
			wantCall:   "ListBudgets",
		},
		{
			name:       "report",
			method:     http.MethodGet,
			path:       "/api/reports/summary?from=2026-11-01&to=2026-11-10",
			wantStatus: http.StatusOK,
			wantBody:   `{"from":"2026-11-01","to":"2026-11-10","total":0,"categories":[]}`,
			wantCall:   "GetReportSummary",
		},
		{
			name:       "missing report dates",
			method:     http.MethodGet,
			path:       "/api/reports/summary",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"from is required and must have format YYYY-MM-DD"}`,
		},
		{
			name:       "reversed report dates",
			method:     http.MethodGet,
			path:       "/api/reports/summary?from=2026-11-10&to=2026-11-01",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"from must not be after to"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &serviceStub{err: tt.serviceErr}
			handler := newGatewayHandler(svc)

			req := httptest.NewRequest(
				tt.method,
				tt.path,
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					recorder.Code,
					tt.wantStatus,
					recorder.Body.String(),
				)
			}

			if got := recorder.Header().Get("Content-Type"); got !=
				"application/json; charset=utf-8" {
				t.Errorf("unexpected Content-Type: %q", got)
			}

			var got, want any

			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if err := json.Unmarshal([]byte(tt.wantBody), &want); err != nil {
				t.Fatalf("decode expected response: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("body = %s, want %s", recorder.Body.String(), tt.wantBody)
			}

			if svc.called != tt.wantCall {
				t.Errorf("service call = %q, want %q", svc.called, tt.wantCall)
			}

			switch tt.name {
			case "create budget":
				if svc.budget.Category != "food" || svc.budget.Limit != 100 {
					t.Errorf("unexpected budget: %+v", svc.budget)
				}

			case "create transaction":
				if svc.tx.Amount != 10 ||
					svc.tx.Category != "food" ||
					svc.tx.Description != "lunch" ||
					svc.tx.Date.Format("2006-01-02") != "2026-11-10" {
					t.Errorf("unexpected transaction: %+v", svc.tx)
				}

			case "report":
				if svc.from.Format("2006-01-02") != "2026-11-01" ||
					svc.to.Format("2006-01-02") != "2026-11-10" {
					t.Errorf("unexpected report dates: %v / %v", svc.from, svc.to)
				}
			}
		})
	}
}
