package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Ledger service started")

	testTransactions := []Transaction{
		{
			Amount:      1250.50,
			Category:    "Продукты",
			Description: "Покупка продуктов шестерочка",
		},
		{
			Amount:      199.00,
			Category:    "Транспорт",
			Description: "Оплата проездного",
		},
		{
			Amount:      399.99,
			Category:    "Обучение",
			Description: "Курс по Go",
		},
	}

	for _, tx := range testTransactions {
		if err := AddTransaction(tx); err != nil {
			log.Printf("failed to add transaction: %v", err)
		}
	}

	transactions := ListTransactions()

	fmt.Println("\nTransactions:")
	for _, tx := range transactions {
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
