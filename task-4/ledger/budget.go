package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
)

type Budget struct {
	Category string
	Limit    float64
	Period   string
}

var (
	transactions = make([]Transaction, 0)
	budgets      = make(map[string]Budget)
	mu           sync.RWMutex
)

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

	mu.Lock()
	defer mu.Unlock()

	budgets[b.Category] = b
	return nil
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

	if err := json.NewDecoder(r).Decode(&loaded); err != nil {
		return fmt.Errorf("failed to parse budgets JSON: %w", err)
	}

	for i, b := range loaded {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("invalid budget at index %d: %w", i, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	for _, b := range loaded {
		budgets[b.Category] = b
	}

	return nil
}
