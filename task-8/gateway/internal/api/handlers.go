package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"ledger/service"
)

type Handler struct {
	service service.Service
}

func New(svc service.Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/budgets", h.CreateBudget)
	mux.HandleFunc("GET /api/budgets", h.ListBudgets)
	mux.HandleFunc("POST /api/transactions", h.CreateTransaction)
	mux.HandleFunc("GET /api/transactions", h.ListTransactions)
	mux.HandleFunc("GET /api/reports/summary", h.ReportSummary)

	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeServiceError(
	w http.ResponseWriter,
	operation string,
	err error,
) {
	var validationErr *service.ValidationError

	switch {
	case errors.As(err, &validationErr):
		writeError(w, http.StatusBadRequest, validationErr.Error())

	case errors.Is(err, service.ErrBudgetExceeded):
		writeError(w, http.StatusConflict, "budget exceeded")

	default:
		log.Printf("%s: %v", operation, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) CreateBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	budget := req.ToDomain()

	if err := h.service.SetBudget(r.Context(), budget); err != nil {
		writeServiceError(w, "SetBudget", err)
		return
	}

	writeJSON(w, http.StatusCreated, FromDomainBudget(budget))
}

func (h *Handler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	budgets, err := h.service.ListBudgets(r.Context())
	if err != nil {
		writeServiceError(w, "ListBudgets", err)
		return
	}

	response := make([]BudgetResponse, 0, len(budgets))

	for _, budget := range budgets {
		response = append(response, FromDomainBudget(budget))
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
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

	created, err := h.service.AddTransaction(r.Context(), tx)
	if err != nil {
		writeServiceError(w, "AddTransaction", err)
		return
	}

	writeJSON(w, http.StatusCreated, FromDomainTransaction(created))
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := h.service.ListTransactions(r.Context())
	if err != nil {
		writeServiceError(w, "ListTransactions", err)
		return
	}

	response := make([]TransactionResponse, 0, len(transactions))

	for _, tx := range transactions {
		response = append(response, FromDomainTransaction(tx))
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) ReportSummary(w http.ResponseWriter, r *http.Request) {
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"from is required and must have format YYYY-MM-DD",
		)
		return
	}

	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"to is required and must have format YYYY-MM-DD",
		)
		return
	}

	if from.After(to) {
		writeError(w, http.StatusBadRequest, "from must not be after to")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	report, err := h.service.GetReportSummary(ctx, from, to)
	if err != nil {
		writeServiceError(w, "GetReportSummary", err)
		return
	}

	writeJSON(w, http.StatusOK, report)
}
