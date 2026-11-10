package service

import (
	"ledger/internal/domain"
	appservice "ledger/service"
)

type Application struct {
	*Service
	*ReportService
}

func NewApplication(
	budgets domain.BudgetRepository,
	transactions domain.TransactionRepository,
	reports domain.ReportRepository,
	cache domain.ReportCache,
) *Application {
	return &Application{
		Service:       New(budgets, transactions),
		ReportService: NewReportService(reports, cache),
	}
}

var _ appservice.Service = (*Application)(nil)
