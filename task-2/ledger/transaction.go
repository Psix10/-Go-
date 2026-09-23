package main

import (
	"errors"
	"sync"
	"time"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

var (
	transactions = make([]Transaction, 0)
	mu           sync.RWMutex
)

func AddTransaction(tx Transaction) error {
	if tx.Amount == 0 {
		return errors.New("transaction amount cannot be zero")
	}

	mu.Lock()
	defer mu.Unlock()

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
