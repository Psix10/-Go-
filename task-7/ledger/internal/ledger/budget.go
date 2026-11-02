package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

type Budget struct {
	Category string
	Limit    float64
	Period   string
}

func (b Budget) Validate() error {
	if strings.TrimSpace(b.Category) == "" {
		return errors.New("budget category cannot be empty")
	}

	if math.IsNaN(b.Limit) || math.IsInf(b.Limit, 0) || b.Limit <= 0 {
		return errors.New("budget limit must be a finite positive number")
	}

	return nil
}

func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid budget: %w", err)
	}

	if database == nil {
		return fmt.Errorf("PostgreSQL is not initialized")
	}

	const query = `
		INSERT INTO budgets (category, limit_amount)
		VALUES ($1, $2)
		ON CONFLICT (category)
		DO UPDATE SET limit_amount = EXCLUDED.limit_amount
	`

	if _, err := database.Exec(query, b.Category, b.Limit); err != nil {
		return fmt.Errorf("save budget: %w", err)
	}

	return nil
}

func ListBudgets() ([]Budget, error) {
	if database == nil {
		return nil, fmt.Errorf("PostgreSQL is not initialized")
	}

	const query = `
		SELECT category, limit_amount
		FROM budgets
		ORDER BY category
	`

	rows, err := database.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query budgets: %w", err)
	}
	defer rows.Close()

	result := make([]Budget, 0)

	for rows.Next() {
		var b Budget

		if err := rows.Scan(&b.Category, &b.Limit); err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}

		b.Period = "month"
		result = append(result, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate budgets: %w", err)
	}

	return result, nil
}

func LoadBudgets(r io.Reader) error {
	var loaded []Budget

	if err := json.NewDecoder(r).Decode(&loaded); err != nil {
		return fmt.Errorf("failed to parse budgets JSON: %w", err)
	}

	// Проверяем весь входной список до начала записи.
	for i, b := range loaded {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("invalid budget at index %d: %w", i, err)
		}
	}

	for i, b := range loaded {
		if err := SetBudget(b); err != nil {
			return fmt.Errorf("save budget at index %d: %w", i, err)
		}
	}

	return nil
}
