package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"gateway/internal/ledger"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	tx, err := req.ToDomain()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := tx.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := ledger.AddTransaction(tx)
	if err != nil {
		if errors.Is(err, ledger.ErrBudgetExceeded) {
			writeError(w, http.StatusConflict, "budget exceeded")
			return
		}
		log.Printf("AddTransaction: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, FromDomainTransaction(created))
}

func ListTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	txs := ledger.ListTransactions()
	resp := make([]TransactionResponse, 0, len(txs))
	for _, t := range txs {
		resp = append(resp, FromDomainTransaction(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

func CreateBudgetHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	b := req.ToDomain()
	if err := b.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := ledger.SetBudget(b); err != nil {
		log.Printf("SetBudget: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, FromDomainBudget(b))
}

func ListBudgetsHandler(w http.ResponseWriter, r *http.Request) {
	budgets := ledger.ListBudgets()
	resp := make([]BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		resp = append(resp, FromDomainBudget(b))
	}
	writeJSON(w, http.StatusOK, resp)
}
