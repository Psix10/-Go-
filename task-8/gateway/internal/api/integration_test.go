//go:build integration

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ledger/app"
	"ledger/service"
)

func resetAPITestLedger(t *testing.T) service.Service {
	t.Helper()

	dsn := PrepareDatabase(
		t,
		"API_TEST_DATABASE_URL",
		"cashapp_api_test",
	)

	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "15")

	svc, closeFn, err := app.New(context.Background())
	if err != nil {
		t.Fatalf("initialize API test service: %v", err)
	}

	t.Cleanup(func() {
		if err := closeFn(); err != nil {
			t.Errorf("close API test service: %v", err)
		}
	})

	return svc
}

func newTestHandler(svc service.Service) http.Handler {
	return New(svc).Routes()
}

func performTestRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	return recorder
}

func requireTestStatus(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedStatus int,
) {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Fatalf(
			"expected status=%d, got=%d; body=%s",
			expectedStatus,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: %q", got)
	}

	if !json.Valid(recorder.Body.Bytes()) {
		t.Fatalf("invalid JSON response: %s", recorder.Body.String())
	}
}

func requireAPIError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedMessage string,
) {
	t.Helper()

	var response struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if !strings.Contains(response.Error, expectedMessage) {
		t.Fatalf(
			"expected error containing %q, got %q",
			expectedMessage,
			response.Error,
		)
	}
}

func createTestBudget(t *testing.T, handler http.Handler) {
	t.Helper()

	recorder := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/budgets",
		`{"category":"food","limit":5000}`,
	)

	requireTestStatus(t, recorder, http.StatusCreated)
}

func TestCreateBudget(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		status  int
		wantErr string
	}{
		{
			"valid",
			`{"category":"food","limit":5000}`,
			http.StatusCreated,
			"",
		},
		{
			"invalid JSON",
			`{"category":`,
			http.StatusBadRequest,
			"invalid JSON",
		},
		{
			"zero limit",
			`{"category":"food","limit":0}`,
			http.StatusBadRequest,
			"budget limit must be a finite positive number",
		},
		{
			"negative limit",
			`{"category":"food","limit":-100}`,
			http.StatusBadRequest,
			"budget limit must be a finite positive number",
		},
		{
			"empty category",
			`{"category":"","limit":100}`,
			http.StatusBadRequest,
			"budget category cannot be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(st *testing.T) {
			svc := resetAPITestLedger(st)

			recorder := performTestRequest(
				st,
				newTestHandler(svc),
				http.MethodPost,
				"/api/budgets",
				tc.body,
			)

			requireTestStatus(st, recorder, tc.status)

			if tc.wantErr != "" {
				requireAPIError(st, recorder, tc.wantErr)
				return
			}

			var result BudgetResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				st.Fatalf("decode budget: %v", err)
			}

			if result.Category != "food" ||
				result.Limit != 5000 ||
				result.Period != "month" {
				st.Fatalf("unexpected budget: %+v", result)
			}
		})
	}
}

func TestListBudgets(t *testing.T) {
	svc := resetAPITestLedger(t)
	handler := newTestHandler(svc)
	createTestBudget(t, handler)

	recorder := performTestRequest(
		t, handler, http.MethodGet, "/api/budgets", "",
	)
	requireTestStatus(t, recorder, http.StatusOK)

	var result []BudgetResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode budgets: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected one budget, got %+v", result)
	}

	if result[0].Category != "food" || result[0].Limit != 5000 {
		t.Fatalf("unexpected budget: %+v", result[0])
	}
}

func TestCreateTransaction(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		status  int
		wantErr string
	}{
		{
			"valid",
			`{"amount":100,"category":"food","description":"test purchase","date":"2024-01-15"}`,
			http.StatusCreated,
			"",
		},
		{
			"exceeds budget",
			`{"amount":6000,"category":"food","date":"2024-01-15"}`,
			http.StatusConflict,
			"budget exceeded",
		},
		{
			"invalid JSON",
			`{"amount":`,
			http.StatusBadRequest,
			"invalid JSON",
		},
		{
			"invalid date",
			`{"amount":100,"category":"food","date":"not-a-date"}`,
			http.StatusBadRequest,
			"invalid date",
		},
		{
			"zero amount",
			`{"amount":0,"category":"food","date":"2024-01-15"}`,
			http.StatusBadRequest,
			"transaction amount must be a finite positive number",
		},
		{
			"negative amount",
			`{"amount":-100,"category":"food","date":"2024-01-15"}`,
			http.StatusBadRequest,
			"transaction amount must be a finite positive number",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(st *testing.T) {
			svc := resetAPITestLedger(st)
			handler := newTestHandler(svc)
			createTestBudget(st, handler)

			recorder := performTestRequest(
				st, handler, http.MethodPost, "/api/transactions", tc.body,
			)
			requireTestStatus(st, recorder, tc.status)

			if tc.wantErr != "" {
				requireAPIError(st, recorder, tc.wantErr)

				txs, err := svc.ListTransactions(context.Background())
				if err != nil {
					st.Fatalf("list transactions: %v", err)
				}
				if len(txs) != 0 {
					st.Fatalf("rejected transaction was saved: %+v", txs)
				}
				return
			}

			var result TransactionResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				st.Fatalf("decode transaction: %v", err)
			}

			if result.ID <= 0 ||
				result.Amount != 100 ||
				result.Category != "food" ||
				result.Description != "test purchase" ||
				result.Date != "2024-01-15" {
				st.Fatalf("unexpected transaction: %+v", result)
			}
		})
	}
}

func TestListTransactions(t *testing.T) {
	svc := resetAPITestLedger(t)
	handler := newTestHandler(svc)
	createTestBudget(t, handler)

	created := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		`{"amount":100,"category":"food","description":"test purchase","date":"2024-01-15"}`,
	)
	requireTestStatus(t, created, http.StatusCreated)

	listed := performTestRequest(
		t, handler, http.MethodGet, "/api/transactions", "",
	)
	requireTestStatus(t, listed, http.StatusOK)

	var result []TransactionResponse
	if err := json.Unmarshal(listed.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode transactions: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected one transaction, got %+v", result)
	}

	if result[0].Amount != 100 ||
		result[0].Category != "food" ||
		result[0].Description != "test purchase" ||
		result[0].Date != "2024-01-15" {
		t.Fatalf("unexpected transaction: %+v", result[0])
	}
}

func TestMainSequence(t *testing.T) {
	svc := resetAPITestLedger(t)
	handler := newTestHandler(svc)
	createTestBudget(t, handler)

	created := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		`{"amount":100,"category":"food","date":"2024-01-15"}`,
	)
	requireTestStatus(t, created, http.StatusCreated)

	before, err := svc.ListTransactions(context.Background())
	if err != nil {
		t.Fatalf("list before rejection: %v", err)
	}

	rejected := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		`{"amount":5000,"category":"food","date":"2024-01-15"}`,
	)
	requireTestStatus(t, rejected, http.StatusConflict)
	requireAPIError(t, rejected, "budget exceeded")

	after, err := svc.ListTransactions(context.Background())
	if err != nil {
		t.Fatalf("list after rejection: %v", err)
	}

	if len(before) != 1 || len(after) != len(before) {
		t.Fatalf(
			"unexpected transaction counts: before=%d, after=%d",
			len(before),
			len(after),
		)
	}

	badJSON := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		"{invalid json",
	)
	requireTestStatus(t, badJSON, http.StatusBadRequest)
	requireAPIError(t, badJSON, "invalid JSON")
}

func TestEmptyLists(t *testing.T) {
	svc := resetAPITestLedger(t)
	handler := newTestHandler(svc)

	for _, path := range []string{"/api/budgets", "/api/transactions"} {
		recorder := performTestRequest(t, handler, http.MethodGet, path, "")
		requireTestStatus(t, recorder, http.StatusOK)

		if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
			t.Fatalf("%s: expected [], got %s", path, got)
		}
	}
}
