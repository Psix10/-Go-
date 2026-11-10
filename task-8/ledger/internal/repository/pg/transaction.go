package pg

import (
	"context"
	"database/sql"
	"fmt"

	"ledger/internal/domain"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

var _ domain.TransactionRepository = (*TransactionRepository)(nil)

func (r *TransactionRepository) Add(
	ctx context.Context,
	tx domain.Transaction,
) (domain.Transaction, error) {
	const query = `
		INSERT INTO expenses (amount, category, description, date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, amount, date
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		tx.Amount,
		tx.Category,
		tx.Description,
		tx.Date.Format("2006-01-02"),
	).Scan(
		&tx.ID,
		&tx.Amount,
		&tx.Date,
	)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}

	return tx, nil
}

func (r *TransactionRepository) List(
	ctx context.Context,
) ([]domain.Transaction, error) {
	const query = `
		SELECT id, amount, category, COALESCE(description, ''), date
		FROM expenses
		ORDER BY date DESC, id DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Transaction, 0)

	for rows.Next() {
		var tx domain.Transaction

		if err := rows.Scan(
			&tx.ID,
			&tx.Amount,
			&tx.Category,
			&tx.Description,
			&tx.Date,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}

		result = append(result, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}

	return result, nil
}

func (r *TransactionRepository) SumByCategory(
	ctx context.Context,
	category string,
) (float64, error) {
	const query = `
		SELECT COALESCE(SUM(amount), 0)
		FROM expenses
		WHERE category = $1
	`

	var total float64

	if err := r.db.QueryRowContext(ctx, query, category).Scan(
		&total,
	); err != nil {
		return 0, fmt.Errorf("calculate category expenses: %w", err)
	}

	return total, nil
}
