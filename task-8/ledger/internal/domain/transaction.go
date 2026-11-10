package domain

import (
	"errors"
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
