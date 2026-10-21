package api

import (
	"fmt"
	"time"

	"gateway/internal/ledger"
)

type CreateTransactionRequest struct {
	Amount float64 `json:"amount"`
	Category string `json:"category"`
	Description string `json:"description"`
	Date string `json:"date"`
}

type TransactionResponse struct {
	ID int `json:"id"`
	Amount float64 `json:"amount"`
	Category string `json:"category"`
	Description string `json:"description"`
	Date string `json:"date"`
}

type CreateBudgetRequest struct {
	Category string `json:"category"`
	Limit float64 `json:"limit"`
}

type BudgetResponse struct {
	Category string `json:"category"`
	Limit float64 `json:"limit"`
	Period string `json:"period"`
}



func (r CreateTransactionRequest) ToDomain() (ledger.Transaction, error) {
	tx := ledger.Transaction{
		Amount: r.Amount,
		Category: r.Category,
		Description: r.Description,
	}
	if r.Date != "" {
		t, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			return ledger.Transaction{}, fmt.Errorf("invalid date %q, expected format YYYY-MM-DD", r.Date)
		}
		tx.Date = t
	}
	return tx, nil
}

func FromDomainTransaction(t ledger.Transaction) TransactionResponse {
	return TransactionResponse{
		ID: t.ID,
		Amount: t.Amount,
		Category: t.Category,
		Description: t.Description,
		Date: t.Date.Format("2006-01-02"),
	}
}

func (r CreateBudgetRequest) ToDomain() ledger.Budget {
	return ledger.Budget{
		Category: r.Category,
		Limit: r.Limit,
		Period: "month",
	}
}

func FromDomainBudget(b ledger.Budget) BudgetResponse {
	return BudgetResponse{
		Category: b.Category,
		Limit: b.Limit,
		Period: b.Period,
	}
}

