package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

type Budget struct {
	Category string
	Limit float64
	Period string
}

var ( 
	transactions = make([]Transaction, 0)
	budgets = make(map[string]Budget)
	mu sync.RWMutex
)


func SetBudget(b Budget) {
	mu.Lock()
	defer mu.Unlock()
	budgets[b.Category] = b
}

func ListBudgets() []Budget {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Budget, 0, len(budgets))
	for _, b := range budgets {
		result = append(result, b)
	}
	return result
}

func LoadBudgets(r io.Reader) error {
	var loaded []Budget
	dec := json.NewDecoder(r)
	if err := dec.Decode(&loaded); err != nil {
		return fmt.Errorf("failed to parse budgets JSON: %w", err)
	}
	for _, b := range loaded {
		if b.Category == "" {
			return errors.New("budget category cannot be empty")
		}
		if b.Limit < 0 {
			return fmt.Errorf("budget limit for %q cannot be negative", b.Category)
		}
		SetBudget(b)
	}
	return nil
}