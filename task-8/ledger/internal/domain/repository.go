package domain

import "context"

type BudgetRepository interface {
	Upsert(ctx context.Context, budget Budget) error
	GetByCategory(ctx context.Context, category string) (Budget, error)
	List(ctx context.Context) ([]Budget, error)
}

type TransactionRepository interface {
	Add(ctx context.Context, tx Transaction) (Transaction, error)
	List(ctx context.Context) ([]Transaction, error)
	SumByCategory(ctx context.Context, category string) (float64, error)
}
