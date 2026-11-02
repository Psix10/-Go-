package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

const serverAddress = ":8080"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ledgerURL := os.Getenv("LEDGER_URL")
	if ledgerURL == "" {
		ledgerURL = "http://127.0.0.1:8081"
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 12 * time.Second,
	}
	defer transport.CloseIdleConnections()

	handler, err := newGatewayHandler(ledgerURL, transport)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              serverAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Gateway service started on http://localhost%s", serverAddress)
	log.Printf("Ledger upstream: %s", ledgerURL)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("Gateway HTTP server failed: %w", err)
	}

	return nil
}

func newGatewayHandler(
	ledgerURL string,
	transport http.RoundTripper,
) (http.Handler, error) {
	target, err := url.Parse(ledgerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid LEDGER_URL: %w", err)
	}

	if (target.Scheme != "http" && target.Scheme != "https") ||
		target.Hostname() == "" ||
		target.User != nil ||
		(target.Path != "" && target.Path != "/") ||
		target.RawQuery != "" ||
		target.Fragment != "" {
		return nil, fmt.Errorf(
			"LEDGER_URL must be an HTTP(S) origin, for example http://127.0.0.1:8081",
		)
	}

	target.Path = ""

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
		},
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Ledger proxy failed: %v", err)

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)

			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "ledger service unavailable",
			})
		},
	}

	mux := http.NewServeMux()

	mux.Handle("POST /api/budgets", proxy)
	mux.Handle("GET /api/budgets", proxy)
	mux.Handle("POST /api/transactions", proxy)
	mux.Handle("GET /api/transactions", proxy)
	mux.Handle("GET /api/reports/summary", proxy)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})

	return handler, nil
}
