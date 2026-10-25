package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type httpTestCase struct {
	name                 string
	method               string
	path                 string
	body                 string
	expectedStatus       int
	expectedBodyContains string
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

			if tc.expectedBodyContains != "" &&
				!strings.Contains(
					recorder.Body.String(),
					tc.expectedBodyContains,
				) {
				st.Errorf(
					"response body doesn't contain expected text:\n"+
						"got: %q\nwant substring: %q",
					recorder.Body.String(),
					tc.expectedBodyContains,
				)
			}
		})
	}
}

func TestCreateBudget(t *testing.T) {
	handler := newTestHandler()
	category := "еда-" + t.Name()

	testCases := []httpTestCase{
		{
			name:   "valid budget creation",
			method: http.MethodPost,
			path:   "/api/budgets",
			body: marshalTestJSON(t, map[string]interface{}{
				"category": category,
				"limit":    5000.0,
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
				"category": category + "-zero",
				"limit":    0.0,
			}),
			expectedStatus: http.StatusBadRequest,
			expectedBodyContains: "budget limit must be " +
				"a finite positive number",
		},
		{
			name:   "negative limit",
			method: http.MethodPost,
			path:   "/api/budgets",
			body: marshalTestJSON(t, map[string]interface{}{
				"category": category + "-negative",
				"limit":    -100.0,
			}),
			expectedStatus: http.StatusBadRequest,
			expectedBodyContains: "budget limit must be " +
				"a finite positive number",
		},
	}

	runHTTPTestCases(t, handler, testCases)
}

func TestListBudgets(t *testing.T) {
	handler := newTestHandler()
	category := "еда-" + t.Name()

	createTestBudget(t, handler, category, 5000)

	recorder := performTestRequest(
		t,
		handler,
		http.MethodGet,
		"/api/budgets",
		"",
	)

	requireTestStatus(t, recorder, http.StatusOK)
	requireTestJSONResponse(t, recorder)

	var budgets []BudgetResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &budgets); err != nil {
		t.Fatalf("failed to parse budget response: %v", err)
	}

	if len(budgets) == 0 {
		t.Fatal("expected at least one budget in response")
	}

	requireTestCategory(t, recorder, category)
}

func TestCreateTransaction(t *testing.T) {
	handler := newTestHandler()
	category := "еда-" + t.Name()

	createTestBudget(t, handler, category, 5000)

	testCases := []httpTestCase{
		{
			name:           "valid transaction creation",
			method:         http.MethodPost,
			path:           "/api/transactions",
			body:           transactionTestBody(t, category, 100),
			expectedStatus: http.StatusCreated,
		},
		{
			name:                 "transaction exceeds budget",
			method:               http.MethodPost,
			path:                 "/api/transactions",
			body:                 transactionTestBody(t, category, 6000),
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
	handler := newTestHandler()
	category := "еда-" + t.Name()

	createTestBudget(t, handler, category, 5000)

	createRecorder := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		transactionTestBody(t, category, 100),
	)

	requireTestStatus(t, createRecorder, http.StatusCreated)
	requireTestJSONResponse(t, createRecorder)

	listRecorder := performTestRequest(
		t,
		handler,
		http.MethodGet,
		"/api/transactions",
		"",
	)

	requireTestStatus(t, listRecorder, http.StatusOK)
	requireTestJSONResponse(t, listRecorder)

	var transactions []TransactionResponse
	if err := json.Unmarshal(
		listRecorder.Body.Bytes(),
		&transactions,
	); err != nil {
		t.Fatalf("failed to parse transaction response: %v", err)
	}

	if len(transactions) == 0 {
		t.Fatal("expected at least one transaction in response")
	}

	requireTestCategory(t, listRecorder, category)
}

func requireTestCategory(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedCategory string,
) {
	t.Helper()

	var items []struct {
		Category string `json:"category"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse response categories: %v", err)
	}

	for _, item := range items {
		if item.Category == expectedCategory {
			return
		}
	}

	t.Fatalf(
		"expected category %q in response, got: %s",
		expectedCategory,
		recorder.Body.String(),
	)
}

func TestMainSequence(t *testing.T) {
	handler := newTestHandler()
	category := "еда-" + t.Name()

	// 1. Создаём бюджет.
	createTestBudget(t, handler, category, 5000)

	// 2. Создаём транзакцию в пределах бюджета.
	createRecorder := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		transactionTestBody(t, category, 100),
	)

	requireTestStatus(t, createRecorder, http.StatusCreated)
	requireTestJSONResponse(t, createRecorder)

	// 3. Проверяем, что созданная транзакция присутствует в списке.
	listRecorder := performTestRequest(
		t,
		handler,
		http.MethodGet,
		"/api/transactions",
		"",
	)

	requireTestStatus(t, listRecorder, http.StatusOK)
	requireTestJSONResponse(t, listRecorder)
	requireTestCategory(t, listRecorder, category)

	// 4. Уже потрачено 100: ещё 5000 должны превысить лимит 5000.
	conflictRecorder := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		transactionTestBody(t, category, 5000),
	)

	requireTestStatus(t, conflictRecorder, http.StatusConflict)
	requireTestJSONResponse(t, conflictRecorder)

	var conflictResponse struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(
		conflictRecorder.Body.Bytes(),
		&conflictResponse,
	); err != nil {
		t.Fatalf("failed to parse conflict response: %v", err)
	}

	if conflictResponse.Error != "budget exceeded" {
		t.Errorf(
			"expected error=%q, got %q",
			"budget exceeded",
			conflictResponse.Error,
		)
	}

	// 5. Проверяем некорректный JSON.
	badRequestRecorder := performTestRequest(
		t,
		handler,
		http.MethodPost,
		"/api/transactions",
		"{invalid json",
	)

	requireTestStatus(t, badRequestRecorder, http.StatusBadRequest)
	requireTestJSONResponse(t, badRequestRecorder)

	if !strings.Contains(
		badRequestRecorder.Body.String(),
		"invalid JSON",
	) {
		t.Errorf(
			"expected invalid JSON error, got: %s",
			badRequestRecorder.Body.String(),
		)
	}
}
