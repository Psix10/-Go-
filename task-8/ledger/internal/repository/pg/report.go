package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ledger/internal/domain"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

var _ domain.ReportRepository = (*ReportRepository)(nil)

func (r *ReportRepository) SumByCategoryBetween(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]domain.CategorySummary, error) {
	const query = `
		SELECT category, SUM(amount)
		FROM expenses
		WHERE date >= $1::date AND date <= $2::date
		GROUP BY category
		ORDER BY category
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("query report summary: %w", err)
	}
	defer rows.Close()

	result := make([]domain.CategorySummary, 0)

	for rows.Next() {
		var item domain.CategorySummary

		if err := rows.Scan(
			&item.Category,
			&item.Amount,
		); err != nil {
			return nil, fmt.Errorf("scan report category: %w", err)
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report categories: %w", err)
	}

	return result, nil
}
