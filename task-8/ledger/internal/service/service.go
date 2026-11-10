package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ledger/internal/domain"
)

type Service struct {
	budgets      domain.BudgetRepository
	transactions domain.TransactionRepository
}

func New(
	budgets domain.BudgetRepository,
	transactions domain.TransactionRepository,
) *Service {
	return &Service{
		budgets:      budgets,
		transactions: transactions,
	}
}

func (s *Service) SetBudget(
	ctx context.Context,
	budget domain.Budget,
) error {
	if err := budget.Validate(); err != nil {
		return fmt.Errorf(
			"invalid budget: %w",
			&domain.ValidationError{Err: err},
		)
	}

	return s.budgets.Upsert(ctx, budget)
}

func (s *Service) ListBudgets(
	ctx context.Context,
) ([]domain.Budget, error) {
	budgets, err := s.budgets.List(ctx)
	if err != nil {
		return nil, err
	}

	if budgets == nil {
		budgets = make([]domain.Budget, 0)
	}

	for i := range budgets {
		budgets[i].Period = "month"
	}

	return budgets, nil
}

func (s *Service) AddTransaction(
	ctx context.Context,
	tx domain.Transaction,
) (domain.Transaction, error) {
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	if err := tx.Validate(); err != nil {
		return domain.Transaction{}, fmt.Errorf(
			"invalid transaction: %w",
			&domain.ValidationError{Err: err},
		)
	}

	budget, err := s.budgets.GetByCategory(ctx, tx.Category)

	switch {
	case errors.Is(err, domain.ErrBudgetNotFound):
		// A category without a budget has no spending limit.

	case err != nil:
		return domain.Transaction{}, err

	default:
		spent, err := s.transactions.SumByCategory(ctx, tx.Category)
		if err != nil {
			return domain.Transaction{}, err
		}

		if spent+tx.Amount > budget.Limit {
			return domain.Transaction{}, fmt.Errorf(
				"%w for %q: total %.2f + amount %.2f > limit %.2f",
				domain.ErrBudgetExceeded,
				tx.Category,
				spent,
				tx.Amount,
				budget.Limit,
			)
		}
	}

	return s.transactions.Add(ctx, tx)
}

func (s *Service) ListTransactions(
	ctx context.Context,
) ([]domain.Transaction, error) {
	transactions, err := s.transactions.List(ctx)
	if err != nil {
		return nil, err
	}

	if transactions == nil {
		transactions = make([]domain.Transaction, 0)
	}

	return transactions, nil
}
