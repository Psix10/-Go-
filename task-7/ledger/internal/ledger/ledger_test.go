package ledger

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"ledger/internal/testutil"
)

func testTransaction(amount float64, category string) Transaction {
	return Transaction{
		Amount:      amount,
		Category:    category,
		Description: "test purchase",
		Date:        time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
	}
}

func resetTestLedger(t *testing.T) {
	t.Helper()

	dsn := testutil.PrepareDatabase(
		t,
		"LEDGER_TEST_DATABASE_URL",
		"cashapp_ledger_test",
	)

	t.Setenv("DATABASE_URL", dsn)

	if err := InitDB(context.Background()); err != nil {
		t.Fatalf("initialize Ledger test database: %v", err)
	}

	t.Cleanup(func() {
		if err := CloseDB(); err != nil {
			t.Errorf("close Ledger test database: %v", err)
		}
	})
}

func requireBudget(t *testing.T, budget Budget) {
	t.Helper()

	if err := SetBudget(budget); err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}
}

func mustListBudgets(t *testing.T) []Budget {
	t.Helper()

	result, err := ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets failed: %v", err)
	}

	return result
}

func mustListTransactions(t *testing.T) []Transaction {
	t.Helper()

	result, err := ListTransactions()
	if err != nil {
		t.Fatalf("ListTransactions failed: %v", err)
	}

	return result
}

func checkLedgerError(
	t *testing.T,
	err error,
	expectedErr bool,
	expectedMessage string,
) {
	t.Helper()

	if !expectedErr {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), expectedMessage) {
		t.Fatalf(
			"unexpected error: got %q, want substring %q",
			err.Error(),
			expectedMessage,
		)
	}
}

func TestTransactionValidation(t *testing.T) {
	t.Parallel()

	const amountError = "transaction amount must be a finite positive number"
	const categoryError = "transaction category cannot be empty"

	cases := []struct {
		name    string
		input   Transaction
		wantErr string
	}{
		{"valid", testTransaction(100, "еда"), ""},
		{"zero amount", testTransaction(0, "еда"), amountError},
		{"negative amount", testTransaction(-100, "еда"), amountError},
		{"NaN amount", testTransaction(math.NaN(), "еда"), amountError},
		{"positive infinity", testTransaction(math.Inf(1), "еда"), amountError},
		{"negative infinity", testTransaction(math.Inf(-1), "еда"), amountError},
		{"empty category", testTransaction(100, ""), categoryError},
		{"whitespace category", testTransaction(100, " \t\n"), categoryError},
		{
			"zero date",
			Transaction{Amount: 100, Category: "еда"},
			"transaction date cannot be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(st *testing.T) {
			st.Parallel()
			checkLedgerError(st, tc.input.Validate(), tc.wantErr != "", tc.wantErr)
		})
	}
}

func TestBudgetValidation(t *testing.T) {
	t.Parallel()

	const limitError = "budget limit must be a finite positive number"
	const categoryError = "budget category cannot be empty"

	cases := []struct {
		name    string
		input   Budget
		wantErr string
	}{
		{"valid", Budget{Category: "еда", Limit: 5000}, ""},
		{"zero limit", Budget{Category: "еда", Limit: 0}, limitError},
		{"negative limit", Budget{Category: "еда", Limit: -100}, limitError},
		{"NaN limit", Budget{Category: "еда", Limit: math.NaN()}, limitError},
		{"positive infinity", Budget{Category: "еда", Limit: math.Inf(1)}, limitError},
		{"negative infinity", Budget{Category: "еда", Limit: math.Inf(-1)}, limitError},
		{"empty category", Budget{Category: "", Limit: 100}, categoryError},
		{"whitespace category", Budget{Category: " \t\n", Limit: 100}, categoryError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(st *testing.T) {
			st.Parallel()
			checkLedgerError(st, tc.input.Validate(), tc.wantErr != "", tc.wantErr)
		})
	}
}

func TestSetBudget(t *testing.T) {
	cases := []struct {
		name    string
		budget  Budget
		wantErr string
	}{
		{"valid", Budget{Category: "еда", Limit: 5000}, ""},
		{
			"zero limit",
			Budget{Category: "еда", Limit: 0},
			"budget limit must be a finite positive number",
		},
		{
			"negative limit",
			Budget{Category: "еда", Limit: -100},
			"budget limit must be a finite positive number",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(st *testing.T) {
			resetTestLedger(st)

			err := SetBudget(tc.budget)
			checkLedgerError(st, err, tc.wantErr != "", tc.wantErr)

			budgets := mustListBudgets(st)

			if tc.wantErr != "" {
				if len(budgets) != 0 {
					st.Fatalf("invalid budget was saved: %+v", budgets)
				}
				return
			}

			if len(budgets) != 1 {
				st.Fatalf("expected one budget, got %d", len(budgets))
			}

			if budgets[0].Category != tc.budget.Category ||
				budgets[0].Limit != tc.budget.Limit {
				st.Fatalf("unexpected saved budget: %+v", budgets[0])
			}
		})
	}
}

func TestSetBudgetUpsert(t *testing.T) {
	resetTestLedger(t)

	requireBudget(t, Budget{Category: "еда", Limit: 5000})
	requireBudget(t, Budget{Category: "еда", Limit: 6000})

	budgets := mustListBudgets(t)

	if len(budgets) != 1 {
		t.Fatalf("expected one budget after UPSERT, got %d", len(budgets))
	}

	if budgets[0].Category != "еда" || budgets[0].Limit != 6000 {
		t.Fatalf("unexpected updated budget: %+v", budgets[0])
	}
}

func TestListBudgets(t *testing.T) {
	resetTestLedger(t)

	requireBudget(t, Budget{Category: "transport", Limit: 1000})
	requireBudget(t, Budget{Category: "food", Limit: 5000})

	budgets := mustListBudgets(t)

	if len(budgets) != 2 {
		t.Fatalf("expected two budgets, got %d", len(budgets))
	}

	if budgets[0].Category != "food" || budgets[0].Limit != 5000 {
		t.Fatalf("unexpected first budget: %+v", budgets[0])
	}

	if budgets[1].Category != "transport" || budgets[1].Limit != 1000 {
		t.Fatalf("unexpected second budget: %+v", budgets[1])
	}

	for _, budget := range budgets {
		if budget.Period != "month" {
			t.Fatalf("expected period month, got %q", budget.Period)
		}
	}
}

func TestAddTransaction(t *testing.T) {
	cases := []struct {
		name    string
		initial float64
		amount  float64
		wantErr bool
	}{
		{"within budget", 0, 100, false},
		{"exact limit", 0, 5000, false},
		{"exceeds budget", 0, 5001, true},
		{"combined expenses exceed budget", 2000, 3001, true},
		{"combined expenses equal budget", 2000, 3000, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(st *testing.T) {
			resetTestLedger(st)
			requireBudget(st, Budget{Category: "еда", Limit: 5000})

			if tc.initial != 0 {
				if _, err := AddTransaction(testTransaction(tc.initial, "еда")); err != nil {
					st.Fatalf("insert initial transaction: %v", err)
				}
			}

			countBefore := len(mustListTransactions(st))

			created, err := AddTransaction(testTransaction(tc.amount, "еда"))

			if tc.wantErr {
				if !errors.Is(err, ErrBudgetExceeded) {
					st.Fatalf("expected ErrBudgetExceeded, got %v", err)
				}

				if got := len(mustListTransactions(st)); got != countBefore {
					st.Fatalf(
						"rejected transaction changed storage: before=%d, after=%d",
						countBefore,
						got,
					)
				}
				return
			}

			if err != nil {
				st.Fatalf("AddTransaction failed: %v", err)
			}

			if created.ID <= 0 || created.Amount != tc.amount {
				st.Fatalf("unexpected created transaction: %+v", created)
			}

			transactions := mustListTransactions(st)
			if len(transactions) != countBefore+1 {
				st.Fatalf("unexpected transaction count: %d", len(transactions))
			}

			for _, saved := range transactions {
				if saved.ID == created.ID {
					if saved.Amount != tc.amount || saved.Category != "еда" {
						st.Fatalf("unexpected saved transaction: %+v", saved)
					}
					return
				}
			}

			st.Fatal("created transaction not found in database")
		})
	}
}

func TestAddTransactionWithoutBudget(t *testing.T) {
	resetTestLedger(t)

	created, err := AddTransaction(testTransaction(100, "no-budget"))
	if err != nil {
		t.Fatalf("transaction without budget must be allowed: %v", err)
	}

	if created.ID <= 0 {
		t.Fatalf("expected generated ID, got %d", created.ID)
	}

	if got := len(mustListTransactions(t)); got != 1 {
		t.Fatalf("expected one transaction, got %d", got)
	}
}

func TestListTransactions(t *testing.T) {
	resetTestLedger(t)
	requireBudget(t, Budget{Category: "еда", Limit: 5000})

	first, err := AddTransaction(testTransaction(100, "еда"))
	if err != nil {
		t.Fatalf("add first transaction: %v", err)
	}

	second, err := AddTransaction(testTransaction(200, "еда"))
	if err != nil {
		t.Fatalf("add second transaction: %v", err)
	}

	transactions := mustListTransactions(t)

	if len(transactions) != 2 {
		t.Fatalf("expected two transactions, got %d", len(transactions))
	}

	if transactions[0].ID != second.ID || transactions[1].ID != first.ID {
		t.Fatalf("unexpected transaction order: %+v", transactions)
	}

	if transactions[0].Amount != 200 ||
		transactions[0].Category != "еда" ||
		transactions[0].Description != "test purchase" ||
		transactions[0].Date.Format("2006-01-02") != "2024-01-15" {
		t.Fatalf("unexpected transaction data: %+v", transactions[0])
	}
}

func TestEmptyLists(t *testing.T) {
	resetTestLedger(t)

	budgets := mustListBudgets(t)
	transactions := mustListTransactions(t)

	if budgets == nil || len(budgets) != 0 {
		t.Fatalf("expected non-nil empty budgets, got %+v", budgets)
	}

	if transactions == nil || len(transactions) != 0 {
		t.Fatalf("expected non-nil empty transactions, got %+v", transactions)
	}
}
