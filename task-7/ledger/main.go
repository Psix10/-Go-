package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"ledger/internal/api"
	"ledger/internal/ledger"
)

const serverAddress = "127.0.0.1:8081"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := ledger.InitDB(context.Background()); err != nil {
		return fmt.Errorf("PostgreSQL initialization failed: %w", err)
	}

	defer func() {
		if err := ledger.CloseDB(); err != nil {
			log.Printf("PostgreSQL close failed: %v", err)
		}
	}()

	log.Println("PostgreSQL connected successfully")

	if err := ledger.InitCache(context.Background()); err != nil {
		return fmt.Errorf("Redis initialization failed: %w", err)
	}

	defer func() {
		if err := ledger.CloseCache(); err != nil {
			log.Printf("Redis close failed: %v", err)
		}
	}()

	log.Println("Redis connected successfully")

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/transactions", api.CreateTransactionHandler)
	mux.HandleFunc("GET /api/transactions", api.ListTransactionsHandler)

	mux.HandleFunc("POST /api/budgets", api.CreateBudgetHandler)
	mux.HandleFunc("GET /api/budgets", api.ListBudgetsHandler)

	mux.HandleFunc("GET /api/reports/summary", api.ReportSummaryHandler)

	server := &http.Server{
		Addr:              serverAddress,
		Handler:           api.LoggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Ledger service started on http://%s", serverAddress)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("Ledger HTTP server failed: %w", err)
	}

	return nil
}
