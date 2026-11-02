package ledger

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

var ErrBudgetExceeded = errors.New("budget exceeded")

func (tx Transaction) Validate() error {
	if math.IsNaN(tx.Amount) || math.IsInf(tx.Amount, 0) || tx.Amount <= 0 {
		return errors.New("transaction amount must be a finite positive number")
	}

	if strings.TrimSpace(tx.Category) == "" {
		return errors.New("transaction category cannot be empty")
	}

	if tx.Date.IsZero() {
		return errors.New("transaction date cannot be empty")
	}

	return nil
}

func AddTransaction(tx Transaction) (Transaction, error) {
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	if err := tx.Validate(); err != nil {
		return Transaction{}, fmt.Errorf("invalid transaction: %w", err)
	}

	if database == nil {
		return Transaction{}, fmt.Errorf("PostgreSQL is not initialized")
	}

	var limit float64

	err := database.QueryRow(
		`SELECT limit_amount FROM budgets WHERE category = $1`,
		tx.Category,
	).Scan(&limit)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Категория без бюджета: добавление разрешено без лимита.

	case err != nil:
		return Transaction{}, fmt.Errorf("get category budget: %w", err)

	default:
		var spent float64

		err := database.QueryRow(
			`SELECT COALESCE(SUM(amount), 0)
			 FROM expenses
			 WHERE category = $1`,
			tx.Category,
		).Scan(&spent)
		if err != nil {
			return Transaction{}, fmt.Errorf("calculate category expenses: %w", err)
		}

		if spent+tx.Amount > limit {
			return Transaction{}, fmt.Errorf(
				"%w for %q: total %.2f + amount %.2f > limit %.2f",
				ErrBudgetExceeded,
				tx.Category,
				spent,
				tx.Amount,
				limit,
			)
		}
	}

	const query = `
		INSERT INTO expenses (amount, category, description, date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, amount, date
	`

	err = database.QueryRow(
		query,
		tx.Amount,
		tx.Category,
		tx.Description,
		tx.Date.Format("2006-01-02"),
	).Scan(&tx.ID, &tx.Amount, &tx.Date)
	if err != nil {
		return Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}

	return tx, nil
}

func ListTransactions() ([]Transaction, error) {
	if database == nil {
		return nil, fmt.Errorf("PostgreSQL is not initialized")
	}

	const query = `
		SELECT id, amount, category, COALESCE(description, ''), date
		FROM expenses
		ORDER BY date DESC, id DESC
	`

	rows, err := database.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	result := make([]Transaction, 0)

	for rows.Next() {
		var tx Transaction

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
