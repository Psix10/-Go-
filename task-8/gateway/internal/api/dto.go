package api

import (
	"fmt"
	"time"

	"ledger/service"
)

type CreateTransactionRequest struct {
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

type TransactionResponse struct {
	ID          int     `json:"id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

type CreateBudgetRequest struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

type BudgetResponse struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period"`
}

func (r CreateTransactionRequest) ToDomain() (service.Transaction, error) {
	if r.Date == "" {
		return service.Transaction{}, fmt.Errorf(
			"transaction date cannot be empty",
		)
	}

	date, err := time.Parse("2006-01-02", r.Date)
	if err != nil {
		return service.Transaction{}, fmt.Errorf(
			"invalid date %q, expected format YYYY-MM-DD",
			r.Date,
		)
	}

	return service.Transaction{
		Amount:      r.Amount,
		Category:    r.Category,
		Description: r.Description,
		Date:        date,
	}, nil
}

func FromDomainTransaction(tx service.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Category:    tx.Category,
		Description: tx.Description,
		Date:        tx.Date.Format("2006-01-02"),
	}
}

func (r CreateBudgetRequest) ToDomain() service.Budget {
	return service.Budget{
		Category: r.Category,
		Limit:    r.Limit,
		Period:   "month",
	}
}

func FromDomainBudget(budget service.Budget) BudgetResponse {
	return BudgetResponse{
		Category: budget.Category,
		Limit:    budget.Limit,
		Period:   budget.Period,
	}
}
