package service

import (
	"context"
	"time"

	"ledger/internal/domain"
)

type ValidationError = domain.ValidationError

type Budget = domain.Budget
type Transaction = domain.Transaction
type CategorySummary = domain.CategorySummary
type ReportSummary = domain.ReportSummary

var ErrBudgetExceeded = domain.ErrBudgetExceeded

type Service interface {
	SetBudget(ctx context.Context, budget Budget) error
	ListBudgets(ctx context.Context) ([]Budget, error)

	AddTransaction(ctx context.Context, tx Transaction) (Transaction, error)
	ListTransactions(ctx context.Context) ([]Transaction, error)

	GetReportSummary(
		ctx context.Context,
		from time.Time,
		to time.Time,
	) (ReportSummary, error)
}

