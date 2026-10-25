package main

import (
	"log"
	"net/http"

	"gateway/internal/api"
	"gateway/internal/ledger"
)

const serverAddress = ":8080"

func main() {
	for _, b := range []ledger.Budget{
		{Category: "Продукты", Limit: 5000, Period: "month"},
		{Category: "Транспорт", Limit: 1000, Period: "month"},
	} {
		if err := ledger.SetBudget(b); err != nil {
			log.Fatalf("failed to set default budget: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/transactions", api.CreateTransactionHandler)
	mux.HandleFunc("GET /api/transactions", api.ListTransactionsHandler)
	mux.HandleFunc("POST /api/budgets", api.CreateBudgetHandler)
	mux.HandleFunc("GET /api/budgets", api.ListBudgetsHandler)

	handler := api.LoggingMiddleware(mux)

	log.Printf("Gateway service started on http://localhost%s", serverAddress)
	if err := http.ListenAndServe(serverAddress, handler); err != nil {
		log.Fatalf("gateway server failed: %v", err)
	}
}
