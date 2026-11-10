package service

import (
	"context"
	"errors"
	"testing"

	"ledger/internal/domain"
)

type budgetStub struct {
	budget   domain.Budget
	getErr   error
	getCalls int
	saved    []domain.Budget
	list     []domain.Budget
}

func (r *budgetStub) Upsert(
	_ context.Context,
	budget domain.Budget,
) error {
	r.saved = append(r.saved, budget)
	return nil
}

func (r *budgetStub) GetByCategory(
	_ context.Context,
	_ string,
) (domain.Budget, error) {
	r.getCalls++
	return r.budget, r.getErr
}

func (r *budgetStub) List(
	_ context.Context,
) ([]domain.Budget, error) {
	return r.list, nil
}

type transactionStub struct {
	total    float64
	sumErr   error
	sumCalls int
	addCalls int
	saved    domain.Transaction
}

func (r *transactionStub) Add(
	_ context.Context,
	tx domain.Transaction,
) (domain.Transaction, error) {
	r.addCalls++
	tx.ID = 42
	r.saved = tx
	return tx, nil
}

func (r *transactionStub) List(
	_ context.Context,
) ([]domain.Transaction, error) {
	return nil, nil
}

func (r *transactionStub) SumByCategory(
	_ context.Context,
	_ string,
) (float64, error) {
	r.sumCalls++
	return r.total, r.sumErr
}

func TestAddTransaction(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	tests := []struct {
		name         string
		amount       float64
		spent        float64
		budgetErr    error
		sumErr       error
		wantErr      error
		wantAnyErr   bool
		wantGetCalls int
		wantSumCalls int
		wantAddCalls int
	}{
		{
			name:         "within limit",
			amount:       20,
			spent:        70,
			wantGetCalls: 1,
			wantSumCalls: 1,
			wantAddCalls: 1,
		},
		{
			name:         "exact limit",
			amount:       30,
			spent:        70,
			wantGetCalls: 1,
			wantSumCalls: 1,
			wantAddCalls: 1,
		},
		{
			name:         "budget exceeded",
			amount:       31,
			spent:        70,
			wantErr:      domain.ErrBudgetExceeded,
			wantGetCalls: 1,
			wantSumCalls: 1,
		},
		{
			name:         "no budget",
			amount:       500,
			budgetErr:    domain.ErrBudgetNotFound,
			wantGetCalls: 1,
			wantAddCalls: 1,
		},
		{
			name:       "invalid amount",
			amount:     -1,
			wantAnyErr: true,
		},
		{
			name:         "budget storage error",
			amount:       10,
			budgetErr:    storageErr,
			wantErr:      storageErr,
			wantGetCalls: 1,
		},
		{
			name:         "sum storage error",
			amount:       10,
			sumErr:       storageErr,
			wantErr:      storageErr,
			wantGetCalls: 1,
			wantSumCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgets := &budgetStub{
				budget: domain.Budget{
					Category: "food",
					Limit:    100,
				},
				getErr: tt.budgetErr,
			}

			transactions := &transactionStub{
				total:  tt.spent,
				sumErr: tt.sumErr,
			}

			svc := New(budgets, transactions)

			tx, err := svc.AddTransaction(
				context.Background(),
				domain.Transaction{
					Amount:   tt.amount,
					Category: "food",
				},
			)

			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
			case tt.wantAnyErr:
				if err == nil {
					t.Fatal("expected validation error")
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tx.ID != 42 {
					t.Errorf("ID = %d, want 42", tx.ID)
				}
				if tx.Date.IsZero() {
					t.Error("missing date was not filled")
				}
			}

			if budgets.getCalls != tt.wantGetCalls {
				t.Errorf(
					"budget lookups = %d, want %d",
					budgets.getCalls,
					tt.wantGetCalls,
				)
			}

			if transactions.sumCalls != tt.wantSumCalls {
				t.Errorf(
					"sum calls = %d, want %d",
					transactions.sumCalls,
					tt.wantSumCalls,
				)
			}

			if transactions.addCalls != tt.wantAddCalls {
				t.Errorf(
					"insert calls = %d, want %d",
					transactions.addCalls,
					tt.wantAddCalls,
				)
			}
		})
	}
}

func TestSetBudget(t *testing.T) {
	budgets := &budgetStub{}
	svc := New(budgets, &transactionStub{})

	ctx := context.Background()

	err := svc.SetBudget(ctx, domain.Budget{
		Category: "food",
		Limit:    100,
	})
	if err != nil {
		t.Fatalf("valid budget: %v", err)
	}

	err = svc.SetBudget(ctx, domain.Budget{
		Category: "food",
		Limit:    0,
	})
	if err == nil {
		t.Fatal("expected invalid budget error")
	}

	if len(budgets.saved) != 1 {
		t.Fatalf("saved budgets = %d, want 1", len(budgets.saved))
	}
}

func TestListBudgets(t *testing.T) {
	budgets := &budgetStub{
		list: []domain.Budget{
			{Category: "food", Limit: 100},
		},
	}
	svc := New(budgets, &transactionStub{})

	result, err := svc.ListBudgets(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatalf("budgets = %d, want 1", len(result))
	}

	if result[0].Period != "month" {
		t.Errorf("period = %q, want month", result[0].Period)
	}
}

func TestEmptyLists(t *testing.T) {
	svc := New(&budgetStub{}, &transactionStub{})
	ctx := context.Background()

	budgets, err := svc.ListBudgets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if budgets == nil || len(budgets) != 0 {
		t.Errorf("expected non-nil empty budgets, got %#v", budgets)
	}

	transactions, err := svc.ListTransactions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if transactions == nil || len(transactions) != 0 {
		t.Errorf("expected non-nil empty transactions, got %#v", transactions)
	}
}
