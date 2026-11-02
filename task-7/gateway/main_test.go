package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGatewayProxy(t *testing.T) {
	cases := []struct {
		name         string
		method       string
		path         string
		requestBody  string
		status       int
		responseBody string
	}{
		{
			name:         "list budgets",
			method:       http.MethodGet,
			path:         "/api/budgets",
			status:       http.StatusOK,
			responseBody: `[{"category":"food","limit":6000,"period":"month"}]`,
		},
		{
			name:         "create budget",
			method:       http.MethodPost,
			path:         "/api/budgets",
			requestBody:  `{"category":"food","limit":6000}`,
			status:       http.StatusCreated,
			responseBody: `{"category":"food","limit":6000,"period":"month"}`,
		},
		{
			name:         "list transactions",
			method:       http.MethodGet,
			path:         "/api/transactions",
			status:       http.StatusOK,
			responseBody: `[]`,
		},
		{
			name:         "create transaction",
			method:       http.MethodPost,
			path:         "/api/transactions",
			requestBody:  `{"amount":100,"category":"food","date":"2026-11-02"}`,
			status:       http.StatusCreated,
			responseBody: `{"id":1,"amount":100,"category":"food","date":"2026-11-02"}`,
		},
		{
			name:         "budget conflict",
			method:       http.MethodPost,
			path:         "/api/transactions",
			requestBody:  `{"amount":9000,"category":"food","date":"2026-11-02"}`,
			status:       http.StatusConflict,
			responseBody: `{"error":"budget exceeded"}`,
		},
		{
			name:         "invalid request",
			method:       http.MethodPost,
			path:         "/api/budgets",
			requestBody:  `{"category":`,
			status:       http.StatusBadRequest,
			responseBody: `{"error":"invalid JSON"}`,
		},
		{
			name:         "report query parameters",
			method:       http.MethodGet,
			path:         "/api/reports/summary?from=2026-11-01&to=2026-11-30",
			status:       http.StatusOK,
			responseBody: `{"total":1000,"categories":[{"category":"food","amount":1000}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			type receivedRequest struct {
				method      string
				uri         string
				body        string
				contentType string
				err         error
			}

			received := make(chan receivedRequest, 1)

			upstream := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)

					received <- receivedRequest{
						method:      r.Method,
						uri:         r.URL.RequestURI(),
						body:        string(body),
						contentType: r.Header.Get("Content-Type"),
						err:         err,
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					_, _ = io.WriteString(w, tc.responseBody)
				},
			))
			defer upstream.Close()

			handler, err := newGatewayHandler(
				upstream.URL,
				upstream.Client().Transport,
			)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(
				tc.method,
				tc.path,
				strings.NewReader(tc.requestBody),
			)
			if tc.requestBody != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tc.status {
				t.Fatalf(
					"expected status=%d, got=%d; body=%s",
					tc.status,
					recorder.Code,
					recorder.Body.String(),
				)
			}

			if recorder.Body.String() != tc.responseBody {
				t.Fatalf("unexpected response: %s", recorder.Body.String())
			}

			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response Content-Type was not preserved")
			}

			select {
			case got := <-received:
				if got.err != nil {
					t.Fatalf("read upstream request: %v", got.err)
				}

				if got.method != tc.method ||
					got.uri != tc.path ||
					got.body != tc.requestBody {
					t.Fatalf("unexpected upstream request: %+v", got)
				}

				if tc.requestBody != "" &&
					got.contentType != "application/json" {
					t.Fatalf("request Content-Type was not preserved")
				}

			default:
				t.Fatal("upstream did not receive the request")
			}
		})
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated Ledger connection failure")
}

func TestGatewayUnavailable(t *testing.T) {
	handler, err := newGatewayHandler(
		"http://127.0.0.1:8081",
		failingTransport{},
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/budgets", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", recorder.Code)
	}

	var response struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if response.Error != "ledger service unavailable" {
		t.Fatalf("unexpected error: %q", response.Error)
	}
}

func TestGatewayInvalidURL(t *testing.T) {
	cases := []string{
		"",
		"localhost:8081",
		"ftp://localhost:8081",
		"http://",
		"http://user:pass@localhost:8081",
		"http://localhost:8081/api",
		"http://localhost:8081?debug=true",
		"http://localhost:8081#fragment",
	}

	for _, value := range cases {
		t.Run(value, func(t *testing.T) {
			if _, err := newGatewayHandler(value, nil); err == nil {
				t.Fatalf("expected URL validation error for %q", value)
			}
		})
	}
}
