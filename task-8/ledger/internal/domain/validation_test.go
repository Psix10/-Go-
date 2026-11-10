package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func testTransaction(amount float64, category string) Transaction {
	return Transaction{
		Amount:      amount,
		Category:    category,
		Description: "test purchase",
		Date:        time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
	}
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
