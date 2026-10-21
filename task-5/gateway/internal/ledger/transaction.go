package ledger

import (
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

var ErrBudgetExceeded = errors.New("budget exceeded")	

func AddTransaction(tx Transaction) (Transaction, error) {
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	if err := tx.Validate(); err != nil {
		return Transaction{}, fmt.Errorf("invalid transaction: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if b, ok := budgets[tx.Category]; ok {
		var total float64
		for _, existing := range transactions {
			if existing.Category == tx.Category {
				total += existing.Amount
			}
		}	
		if total+tx.Amount > b.Limit {
			return Transaction{}, fmt.Errorf(
				"%w for %q: total %.2f + amount %.2f > limit %.2f",
				ErrBudgetExceeded, tx.Category, total, tx.Amount, b.Limit,
			)
		}
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)

	return tx, nil
}

func ListTransactions() []Transaction {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Transaction, len(transactions))
	copy(result, transactions)

	return result
}
