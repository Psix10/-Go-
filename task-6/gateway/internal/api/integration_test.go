package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gateway/internal/ledger"
)

type httpTestCase struct {
	name                 string
	method               string
	path                 string
	body                 string
	expectedStatus       int
	expectedBodyContains string
}

func resetAPITestLedger(t *testing.T) {
	t.Helper()

	ledger.Reset()
	t.Cleanup(ledger.Reset)
}

func newTestHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/budgets", CreateBudgetHandler)
	mux.HandleFunc("GET /api/budgets", ListBudgetsHandler)
	mux.HandleFunc("POST /api/transactions", CreateTransactionHandler)
	mux.HandleFunc("GET /api/transactions", ListTransactionsHandler)

	return LoggingMiddleware(mux)
}

func marshalTestJSON(t *testing.T, value interface{}) string {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	return string(data)
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
			"expected status=%d, got status=%d\nresponse body: %s",
			expectedStatus,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func requireTestJSONResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) {
	t.Helper()

	const expectedContentType = "application/json; charset=utf-8"

	if got := recorder.Header().Get("Content-Type"); got != expectedContentType {
		t.Errorf(
			"expected Content-Type=%q, got %q",
			expectedContentType,
			got,
		)
	}

	if !json.Valid(recorder.Body.Bytes()) {
		t.Fatalf(
			"expected valid JSON response, got: %q",
			recorder.Body.String(),
		)
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
		t.Fatalf("failed to parse error response: %v", err)
	}

	if !strings.Contains(response.Error, expectedMessage) {
		t.Fatalf(
			"unexpected error response:\ngot: %q\nwant substring: %q",
			response.Error,
			expectedMessage,
		)
	}
}

func createTestBudget(
	t *testing.T,
	handler http.Handler,
	category string,
	limit float64,
) {
	t.Helper()

	body := marshalTestJSON(t, map[string]interface{}{
		"category": category,
		"limit":    limit,
	})

	recorder := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/budgets",
		body,
	)

	requireTestStatus(t, recorder, http.StatusCreated)
	requireTestJSONResponse(t, recorder)

	var response struct {
		Category string  `json:"category"`
		Limit    float64 `json:"limit"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse created budget: %v", err)
	}

	if response.Category != category || response.Limit != limit {
		t.Fatalf(
			"unexpected created budget: got %+v, want category=%q limit=%v",
			response,
			category,
			limit,
		)
	}
}

func transactionTestBody(
	t *testing.T,
	category string,
	amount float64,
) string {
	t.Helper()

	return marshalTestJSON(t, map[string]interface{}{
		"amount":      amount,
		"category":    category,
		"description": "покупка продуктов",
		"date":        "2024-01-15",
	})
}

func requireCreatedTransaction(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	category string,
	amount float64,
) {
	t.Helper()

	requireTestStatus(t, recorder, http.StatusCreated)
	requireTestJSONResponse(t, recorder)

	var response struct {
		Category    string  `json:"category"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse created transaction: %v", err)
	}

	if response.Category != category || response.Amount != amount {
		t.Fatalf(
			"unexpected created transaction: got %+v, want category=%q amount=%v",
			response,
			category,
			amount,
		)
	}

	if response.Description != "покупка продуктов" {
		t.Errorf("unexpected description: %q", response.Description)
	}
}

func requireListedBudget(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	category string,
	limit float64,
) {
	t.Helper()

	requireTestStatus(t, recorder, http.StatusOK)
	requireTestJSONResponse(t, recorder)

	var budgets []struct {
		Category string  `json:"category"`
		Limit    float64 `json:"limit"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &budgets); err != nil {
		t.Fatalf("failed to parse budgets: %v", err)
	}

	for _, budget := range budgets {
		if budget.Category == category && budget.Limit == limit {
			return
		}
	}

	t.Fatalf(
		"budget category=%q limit=%v not found in response: %s",
		category,
		limit,
		recorder.Body.String(),
	)
}

func requireListedTransaction(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	category string,
	amount float64,
) {
	t.Helper()

	requireTestStatus(t, recorder, http.StatusOK)
	requireTestJSONResponse(t, recorder)

	var transactions []struct {
		Category    string  `json:"category"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &transactions); err != nil {
		t.Fatalf("failed to parse transactions: %v", err)
	}

	for _, transaction := range transactions {
		if transaction.Category == category &&
			transaction.Amount == amount &&
			transaction.Description == "покупка продуктов" {
			return
		}
	}

	t.Fatalf(
		"transaction category=%q amount=%v not found in response: %s",
		category,
		amount,
		recorder.Body.String(),
	)
}

func runHTTPTestCases(
	t *testing.T,
	handler http.Handler,
	testCases []httpTestCase,
) {
	t.Helper()

	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			recorder := performTestRequest(
				st,
				handler,
				tc.method,
				tc.path,
				tc.body,
			)

			requireTestStatus(st, recorder, tc.expectedStatus)
			requireTestJSONResponse(st, recorder)

			if tc.expectedBodyContains != "" {
				requireAPIError(st, recorder, tc.expectedBodyContains)
			}
		})
	}
}

func TestCreateBudget(t *testing.T) {
	resetAPITestLedger(t)

	handler := newTestHandler()

	testCases := []httpTestCase{
		{
			name:   "valid budget creation",
			method: http.MethodPost,
			path:   "/api/budgets",
			body: marshalTestJSON(t, map[string]interface{}{
				"category": "еда",
				"limit":    5000,
			}),
			expectedStatus: http.StatusCreated,
		},
		{
			name:                 "invalid JSON",
			method:               http.MethodPost,
			path:                 "/api/budgets",
			body:                 `{"category":`,
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "invalid JSON",
		},
		{
			name:   "zero limit",
			method: http.MethodPost,
			path:   "/api/budgets",
			body: marshalTestJSON(t, map[string]interface{}{
				"category": "еда-zero",
				"limit":    0,
			}),
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "budget limit must be a finite positive number",
		},
		{
			name:   "negative limit",
			method: http.MethodPost,
			path:   "/api/budgets",
			body: marshalTestJSON(t, map[string]interface{}{
				"category": "еда-negative",
				"limit":    -100,
			}),
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "budget limit must be a finite positive number",
		},
	}

	runHTTPTestCases(t, handler, testCases)
}

func TestListBudgets(t *testing.T) {
	resetAPITestLedger(t)

	handler := newTestHandler()
	createTestBudget(t, handler, "еда", 5000)

	recorder := performTestRequest(
		t,
		handler,
		http.MethodGet,
		"/api/budgets",
		"",
	)

	requireListedBudget(t, recorder, "еда", 5000)
}

func TestCreateTransaction(t *testing.T) {
	resetAPITestLedger(t)

	handler := newTestHandler()
	createTestBudget(t, handler, "еда", 5000)

	testCases := []httpTestCase{
		{
			name:           "valid transaction creation",
			method:         http.MethodPost,
			path:           "/api/transactions",
			body:           transactionTestBody(t, "еда", 100),
			expectedStatus: http.StatusCreated,
		},
		{
			name:                 "transaction exceeds budget",
			method:               http.MethodPost,
			path:                 "/api/transactions",
			body:                 transactionTestBody(t, "еда", 6000),
			expectedStatus:       http.StatusConflict,
			expectedBodyContains: "budget exceeded",
		},
		{
			name:                 "invalid JSON",
			method:               http.MethodPost,
			path:                 "/api/transactions",
			body:                 `{"amount":`,
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "invalid JSON",
		},
	}

	runHTTPTestCases(t, handler, testCases)
}

func TestListTransactions(t *testing.T) {
	resetAPITestLedger(t)

	handler := newTestHandler()
	createTestBudget(t, handler, "еда", 5000)

	createRecorder := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		transactionTestBody(t, "еда", 100),
	)

	requireCreatedTransaction(t, createRecorder, "еда", 100)

	listRecorder := performTestRequest(
		t,
		handler,
		http.MethodGet,
		"/api/transactions",
		"",
	)

	requireListedTransaction(t, listRecorder, "еда", 100)
}

func TestMainSequence(t *testing.T) {
	resetAPITestLedger(t)

	handler := newTestHandler()
	const category = "еда"

	// Если подготовительный шаг упал, зависимые шаги не запускаем.
	if !t.Run("create budget", func(st *testing.T) {
		createTestBudget(st, handler, category, 5000)
	}) {
		return
	}

	if !t.Run("create transaction within budget", func(st *testing.T) {
		recorder := performTestRequest(
			st,
			handler,
			http.MethodPost,
			"/api/transactions",
			transactionTestBody(st, category, 100),
		)

		requireCreatedTransaction(st, recorder, category, 100)
	}) {
		return
	}

	t.Run("list transactions", func(st *testing.T) {
		recorder := performTestRequest(
			st,
			handler,
			http.MethodGet,
			"/api/transactions",
			"",
		)

		requireListedTransaction(st, recorder, category, 100)
	})

	t.Run("transaction exceeds budget", func(st *testing.T) {
		countBefore := len(ledger.ListTransactions())

		// Уже потрачено 100: ещё 5000 превышают лимит 5000.
		recorder := performTestRequest(
			st,
			handler,
			http.MethodPost,
			"/api/transactions",
			transactionTestBody(st, category, 5000),
		)

		requireTestStatus(st, recorder, http.StatusConflict)
		requireTestJSONResponse(st, recorder)

		var response struct {
			Error string `json:"error"`
		}

		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			st.Fatalf("failed to parse conflict response: %v", err)
		}

		if response.Error != "budget exceeded" {
			st.Errorf(
				"expected error=%q, got %q",
				"budget exceeded",
				response.Error,
			)
		}

		countAfter := len(ledger.ListTransactions())

		if countAfter != countBefore {
			st.Errorf(
				"rejected transaction changed storage: before=%d, after=%d",
				countBefore,
				countAfter,
			)
		}
	})

	t.Run("bad JSON", func(st *testing.T) {
		recorder := performTestRequest(
			st,
			handler,
			http.MethodPost,
			"/api/transactions",
			"{invalid json",
		)

		requireTestStatus(st, recorder, http.StatusBadRequest)
		requireTestJSONResponse(st, recorder)
		requireAPIError(st, recorder, "invalid JSON")
	})
}
