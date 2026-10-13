package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	fmt.Println("Ledger service started")

	defaultBudgets := []Budget{
		{Category: "Продукты", Limit: 5000, Period: "month"},
		{Category: "Транспорт", Limit: 1000, Period: "month"},
	}

	for _, b := range defaultBudgets {
		if err := SetBudget(b); err != nil {
			log.Fatalf("failed to set default budget: %v", err)
		}
	}

	if f, err := os.Open("budgets.json"); err == nil {
		if err := LoadBudgets(f); err != nil {
			log.Printf("failed to load budgets: %v", err)
		} else {
			fmt.Println("Budgets loaded from budgets.json")
		}

		if err := f.Close(); err != nil {
			log.Printf("failed to close budgets.json: %v", err)
		}
	} else if os.IsNotExist(err) {
		fmt.Println("budgets.json not found, using built-in defaults")
	} else {
		log.Printf("failed to open budgets.json: %v", err)
	}

	fmt.Println("\nChecking Validatable:")

	checks := []struct {
		name  string
		value Validatable
	}{
		{
			name: "Valid transaction",
			value: Transaction{
				Amount:   100,
				Category: "Продукты",
				Date:     time.Now(),
			},
		},
		{
			name: "Invalid transaction",
			value: Transaction{
				Amount:   0,
				Category: "Продукты",
				Date:     time.Now(),
			},
		},
		{
			name: "Valid budget",
			value: Budget{
				Category: "Обучение",
				Limit:    2000,
				Period:   "month",
			},
		},
		{
			name: "Invalid budget",
			value: Budget{
				Category: "Обучение",
				Limit:    0,
				Period:   "month",
			},
		},
	}

	for _, check := range checks {
		if err := CheckValid(check.value); err != nil {
			log.Printf("%s: %v", check.name, err)
		} else {
			fmt.Printf("%s: OK\n", check.name)
		}
	}

	fmt.Println("\nBudgets:")
	for _, b := range ListBudgets() {
		fmt.Printf(
			"Category: %s | Limit: %.2f | Period: %s\n",
			b.Category, b.Limit, b.Period,
		)
	}

	scenarios := []struct {
		tx          Transaction
		expectError bool
	}{
		{
			Transaction{
				Amount:      1250.50,
				Category:    "Продукты",
				Description: "Покупка продуктов",
			},
			false,
		},
		{
			Transaction{
				Amount:      199,
				Category:    "Транспорт",
				Description: "Оплата проездного",
			},
			false,
		},
		{
			Transaction{
				Amount:      399.99,
				Category:    "Обучение",
				Description: "Курс по Go",
			},
			false,
		},
		{
			Transaction{
				Amount:      4000,
				Category:    "Продукты",
				Description: "Крупная закупка",
			},
			true,
		},
		{
			Transaction{Amount: 0, Category: "Продукты"},
			true,
		},
		{
			Transaction{Amount: -100, Category: "Продукты"},
			true,
		},
		{
			Transaction{Amount: 100, Category: "   "},
			true,
		},
	}

	fmt.Println("\nAdding transactions:")
	for i, sc := range scenarios {
		err := AddTransaction(sc.tx)

		if err != nil {
			log.Printf("[scenario %d] отказано: %v", i+1, err)
		} else {
			fmt.Printf(
				"[scenario %d] добавлено: %s %.2f\n",
				i+1, sc.tx.Category, sc.tx.Amount,
			)
		}

		if (err != nil) != sc.expectError {
			log.Printf("[scenario %d] ОШИБКА: неожиданный результат", i+1)
		}
	}

	fmt.Println("\nTransactions in ledger:")
	for _, tx := range ListTransactions() {
		fmt.Printf(
			"ID: %d | Amount: %.2f | Category: %s | Description: %s | Date: %s\n",
			tx.ID,
			tx.Amount,
			tx.Category,
			tx.Description,
			tx.Date.Format("2006-01-02 15:04:05"),
		)
	}
}
