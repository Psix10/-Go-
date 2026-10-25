package ledger

import (
	"math"
	"strings"
	"testing"
	"time"
)

func testTransaction(amount float64, category string) Transaction {
	return Transaction{
		Amount:   amount,
		Category: category,
		Date:     time.Date(2024, time.January, 15, 12, 0, 0, 0, time.UTC),
	}
}

func resetTestLedger(t *testing.T) {
	t.Helper()

	Reset()
	t.Cleanup(Reset)
}

func requireBudget(t *testing.T, budget Budget) {
	t.Helper()

	if err := SetBudget(budget); err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}
}

func checkLedgerError(
	t *testing.T,
	err error,
	expectedErr bool,
	expectedMessage string,
) {
	t.Helper()

	if expectedErr {
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), expectedMessage) {
			t.Fatalf(
				"unexpected error:\ngot: %q\nwant substring: %q",
				err.Error(),
				expectedMessage,
			)
		}

		return
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReset(t *testing.T) {
	resetTestLedger(t)

	requireBudget(t, Budget{
		Category: "еда",
		Limit:    5000,
	})

	requireBudget(t, Budget{
		Category: "транспорт",
		Limit:    1000,
	})

	if _, err := AddTransaction(testTransaction(100, "еда")); err != nil {
		t.Fatalf("failed to add transaction before Reset: %v", err)
	}

	if len(ListBudgets()) == 0 {
		t.Fatal("expected budgets before Reset")
	}

	if len(ListTransactions()) == 0 {
		t.Fatal("expected transactions before Reset")
	}

	Reset()

	if budgets := ListBudgets(); len(budgets) != 0 {
		t.Errorf(
			"expected empty budgets after Reset, got %d items",
			len(budgets),
		)
	}

	if transactions := ListTransactions(); len(transactions) != 0 {
		t.Errorf(
			"expected empty transactions after Reset, got %d items",
			len(transactions),
		)
	}
}

func TestTransactionValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       Transaction
		expectedErr bool
		errMsg      string
	}{
		{
			name:  "valid transaction",
			input: testTransaction(100, "еда"),
		},
		{
			name:        "zero amount",
			input:       testTransaction(0, "еда"),
			expectedErr: true,
			errMsg:      "transaction amount must be a finite positive number",
		},
		{
			name:        "negative amount",
			input:       testTransaction(-100, "еда"),
			expectedErr: true,
			errMsg:      "transaction amount must be a finite positive number",
		},
		{
			name:        "NaN amount",
			input:       testTransaction(math.NaN(), "еда"),
			expectedErr: true,
			errMsg:      "transaction amount must be a finite positive number",
		},
		{
			name:        "positive infinity amount",
			input:       testTransaction(math.Inf(1), "еда"),
			expectedErr: true,
			errMsg:      "transaction amount must be a finite positive number",
		},
		{
			name:        "empty category",
			input:       testTransaction(100, ""),
			expectedErr: true,
			errMsg:      "transaction category cannot be empty",
		},
		{
			name:        "whitespace category",
			input:       testTransaction(100, " \t\n"),
			expectedErr: true,
			errMsg:      "transaction category cannot be empty",
		},
		{
			name: "zero date",
			input: Transaction{
				Amount:   100,
				Category: "еда",
			},
			expectedErr: true,
			errMsg:      "transaction date cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			err := tc.input.Validate()
			checkLedgerError(st, err, tc.expectedErr, tc.errMsg)
		})
	}
}

func TestBudgetValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       Budget
		expectedErr bool
		errMsg      string
	}{
		{
			name:  "valid budget",
			input: Budget{Category: "еда", Limit: 5000},
		},
		{
			name:        "zero limit",
			input:       Budget{Category: "еда", Limit: 0},
			expectedErr: true,
			errMsg:      "budget limit must be a finite positive number",
		},
		{
			name:        "negative limit",
			input:       Budget{Category: "еда", Limit: -100},
			expectedErr: true,
			errMsg:      "budget limit must be a finite positive number",
		},
		{
			name:        "NaN limit",
			input:       Budget{Category: "еда", Limit: math.NaN()},
			expectedErr: true,
			errMsg:      "budget limit must be a finite positive number",
		},
		{
			name:        "positive infinity limit",
			input:       Budget{Category: "еда", Limit: math.Inf(1)},
			expectedErr: true,
			errMsg:      "budget limit must be a finite positive number",
		},
		{
			name:        "empty category",
			input:       Budget{Category: "", Limit: 5000},
			expectedErr: true,
			errMsg:      "budget category cannot be empty",
		},
		{
			name:        "whitespace category",
			input:       Budget{Category: " \t\n", Limit: 5000},
			expectedErr: true,
			errMsg:      "budget category cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			err := tc.input.Validate()
			checkLedgerError(st, err, tc.expectedErr, tc.errMsg)
		})
	}
}

func TestAddTransaction(t *testing.T) {
	testCases := []struct {
		name        string
		initial     []Transaction
		input       Transaction
		expectedErr bool
		errMsg      string
	}{
		{
			name:  "valid transaction within budget",
			input: testTransaction(100, "еда"),
		},
		{
			name:        "transaction exceeds budget",
			input:       testTransaction(5001, "еда"),
			expectedErr: true,
			errMsg:      "budget exceeded",
		},
		{
			name: "multiple transactions exceed budget",
			initial: []Transaction{
				testTransaction(2000, "еда"),
			},
			input:       testTransaction(3001, "еда"),
			expectedErr: true,
			errMsg:      "budget exceeded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			resetTestLedger(st)

			requireBudget(st, Budget{
				Category: "еда",
				Limit:    5000,
			})

			for _, initialTx := range tc.initial {
				if _, err := AddTransaction(initialTx); err != nil {
					st.Fatalf(
						"failed to add initial transaction: %v",
						err,
					)
				}
			}

			countBefore := len(ListTransactions())

			createdTx, err := AddTransaction(tc.input)
			checkLedgerError(st, err, tc.expectedErr, tc.errMsg)

			transactions := ListTransactions()

			if tc.expectedErr {
				if len(transactions) != countBefore {
					st.Errorf(
						"rejected transaction changed storage: before=%d, after=%d",
						countBefore,
						len(transactions),
					)
				}

				return
			}

			if createdTx.Amount != tc.input.Amount {
				st.Errorf(
					"expected amount=%v, got %v",
					tc.input.Amount,
					createdTx.Amount,
				)
			}

			if createdTx.Category != tc.input.Category {
				st.Errorf(
					"expected category=%q, got %q",
					tc.input.Category,
					createdTx.Category,
				)
			}

			if len(transactions) != countBefore+1 {
				st.Fatalf(
					"expected %d transactions, got %d",
					countBefore+1,
					len(transactions),
				)
			}

			for _, savedTx := range transactions {
				if savedTx.ID == createdTx.ID &&
					savedTx.Category == tc.input.Category &&
					savedTx.Amount == tc.input.Amount {
					return
				}
			}

			st.Error("created transaction not found in storage")
		})
	}
}

func TestSetBudget(t *testing.T) {
	testCases := []struct {
		name        string
		input       Budget
		expectedErr bool
		errMsg      string
	}{
		{
			name:  "valid budget",
			input: Budget{Category: "еда", Limit: 5000},
		},
		{
			name:        "zero limit",
			input:       Budget{Category: "еда", Limit: 0},
			expectedErr: true,
			errMsg:      "budget limit must be a finite positive number",
		},
		{
			name:        "negative limit",
			input:       Budget{Category: "еда", Limit: -100},
			expectedErr: true,
			errMsg:      "budget limit must be a finite positive number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			resetTestLedger(st)

			err := SetBudget(tc.input)
			checkLedgerError(st, err, tc.expectedErr, tc.errMsg)

			budgets := ListBudgets()

			if tc.expectedErr {
				if len(budgets) != 0 {
					st.Errorf(
						"invalid budget was saved: got %d budgets",
						len(budgets),
					)
				}

				return
			}

			if len(budgets) != 1 {
				st.Fatalf("expected 1 budget, got %d", len(budgets))
			}

			if budgets[0].Category != tc.input.Category ||
				budgets[0].Limit != tc.input.Limit {
				st.Errorf(
					"saved budget mismatch: got %+v, want %+v",
					budgets[0],
					tc.input,
				)
			}
		})
	}
}

func TestListBudgets(t *testing.T) {
	resetTestLedger(t)

	expected := map[string]float64{
		"еда":       5000,
		"транспорт": 1000,
	}

	for category, limit := range expected {
		requireBudget(t, Budget{
			Category: category,
			Limit:    limit,
			Period:   "month",
		})
	}

	budgets := ListBudgets()

	if len(budgets) != len(expected) {
		t.Fatalf(
			"expected %d budgets, got %d",
			len(expected),
			len(budgets),
		)
	}

	for _, budget := range budgets {
		expectedLimit, exists := expected[budget.Category]
		if !exists {
			t.Fatalf("unexpected budget category: %q", budget.Category)
		}

		if budget.Limit != expectedLimit {
			t.Errorf(
				"category %q: expected limit=%v, got %v",
				budget.Category,
				expectedLimit,
				budget.Limit,
			)
		}

		if budget.Period != "month" {
			t.Errorf(
				"category %q: expected period=month, got %q",
				budget.Category,
				budget.Period,
			)
		}

		delete(expected, budget.Category)
	}

	if len(expected) != 0 {
		t.Errorf("missing budgets: %v", expected)
	}
}

func TestListTransactions(t *testing.T) {
	resetTestLedger(t)

	requireBudget(t, Budget{
		Category: "еда",
		Limit:    5000,
	})

	createdTx, err := AddTransaction(testTransaction(100, "еда"))
	if err != nil {
		t.Fatalf("unexpected error adding transaction: %v", err)
	}

	transactions := ListTransactions()

	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}

	savedTx := transactions[0]

	if savedTx.ID != createdTx.ID {
		t.Errorf(
			"transaction ID mismatch: expected %v, got %v",
			createdTx.ID,
			savedTx.ID,
		)
	}

	if savedTx.Amount != createdTx.Amount {
		t.Errorf(
			"transaction amount mismatch: expected %v, got %v",
			createdTx.Amount,
			savedTx.Amount,
		)
	}

	if savedTx.Category != createdTx.Category {
		t.Errorf(
			"transaction category mismatch: expected %q, got %q",
			createdTx.Category,
			savedTx.Category,
		)
	}
}
