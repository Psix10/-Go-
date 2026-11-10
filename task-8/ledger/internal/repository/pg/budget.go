package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ledger/internal/domain"
)

type BudgetRepository struct {
	db *sql.DB
}

func NewBudgetRepository(db *sql.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

var _ domain.BudgetRepository = (*BudgetRepository)(nil)

func (r *BudgetRepository) Upsert(
	ctx context.Context,
	budget domain.Budget,
) error {
	const query = `
		INSERT INTO budgets (category, limit_amount)
		VALUES ($1, $2)
		ON CONFLICT (category)
		DO UPDATE SET limit_amount = EXCLUDED.limit_amount
	`

	if _, err := r.db.ExecContext(
		ctx,
		query,
		budget.Category,
		budget.Limit,
	); err != nil {
		return fmt.Errorf("save budget: %w", err)
	}

	return nil
}

func (r *BudgetRepository) GetByCategory(
	ctx context.Context,
	category string,
) (domain.Budget, error) {
	const query = `
		SELECT category, limit_amount
		FROM budgets
		WHERE category = $1
	`

	var budget domain.Budget

	err := r.db.QueryRowContext(ctx, query, category).Scan(
		&budget.Category,
		&budget.Limit,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.Budget{}, domain.ErrBudgetNotFound
	}

	if err != nil {
		return domain.Budget{}, fmt.Errorf("get category budget: %w", err)
	}

	return budget, nil
}

func (r *BudgetRepository) List(
	ctx context.Context,
) ([]domain.Budget, error) {
	const query = `
		SELECT category, limit_amount
		FROM budgets
		ORDER BY category
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query budgets: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Budget, 0)

	for rows.Next() {
		var budget domain.Budget

		if err := rows.Scan(
			&budget.Category,
			&budget.Limit,
		); err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}

		result = append(result, budget)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate budgets: %w", err)
	}

	return result, nil
}
