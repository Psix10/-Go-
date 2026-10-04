package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("Ledger service started")

	SetBudget(Budget{Category: "Продукты", Limit: 5000, Period: "month"})
	SetBudget(Budget{Category: "Транспорт", Limit: 1000, Period: "month"})

	if f, err := os.Open("budgets.json"); err == nil {
		defer f.Close()
		if err := LoadBudgets(bufio.NewReader(f)); err != nil {
			log.Printf("failed to load budgest: %v", err)
		} else {
			fmt.Println("Budgets loaded from budgets.json")
			} 
	} else {
		fmt.Println("budgets.json not found, using built-in defaults")
	}

	fmt.Println("\nBudgets:")
	for _, b := range ListBudgets() {
		fmt.Printf("Category: %s | Limit: %.2f | Period %s\n",
					b.Category, b.Limit, b.Period)
	}

	scenarios := []struct {
		tx Transaction
		expectError bool
	}{
		{Transaction{Amount: 1250.50, Category: "Продукты", Description: "Покупка продуктов шестерочка"}, false},
		{Transaction{Amount: 199.00, Category: "Транспорт", Description: "Оплата проездного"}, false},
		// Категория без бюджета — ограничений нет.
		{Transaction{Amount: 399.99, Category: "Обучение", Description: "Курс по Go"}, false},
		// Превышение лимита по «Продукты» (5000): 1250.50 + 4000 > 5000.
		{Transaction{Amount: 4000.00, Category: "Продукты", Description: "Крупная закупка"}, true},
	}
	
	fmt.Println("\nAdding transaction:")
	for i, sc := range scenarios {
		err := AddTransaction(sc.tx)
		if err != nil {
			log.Printf("[scenario %d] отказано (%s %.2f): %v", i+1, sc.tx.Category, sc.tx.Amount, err)
			if !sc.expectError {
				log.Printf("[scenario %d] ОШИБКА: ожидался успех", i+1)
			}
		} else {
			fmt.Printf("[scenario %d] добавлено: %s %.2f\n", i+1, sc.tx.Category, sc.tx.Amount)
			if sc.expectError {
				log.Printf("[scenario %d] ОШИБКА: ожидалась ошибка бюджета", i+1)
			}
		}
	}

	fmt.Println("\nTransaction in ledger:")
	for _, tx := range ListTransactions() {
		fmt.Printf(
			"ID: %d | Amount: %.2f | Category: %s | Description %s | Date: %s \n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date.Format("2006-01-02 15:04:05"),
		)
	}

}
