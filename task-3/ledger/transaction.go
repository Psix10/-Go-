package main

import (
	"errors"
	"time"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}


func AddTransaction(tx Transaction) error {
	if tx.Amount == 0 {
		return errors.New("transaction amount cannot be zero")
	}
	if tx.Amount < 0 {
		return errors.New("transaction amount cannot be negative")
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
		if total + tx.Amount > b.Limit {
			return  errors.New("budget exceeded")
		}
	}

	tx.ID = len(transactions) + 1

	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	transactions = append(transactions, tx)

	return nil
}

func ListTransactions() []Transaction {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Transaction, len(transactions))
	copy(result, transactions)

	return result
}
