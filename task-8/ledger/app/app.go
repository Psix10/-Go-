package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ledger/internal/repository/pg"
	rediscache "ledger/internal/repository/redis"
	internalservice "ledger/internal/service"
	"ledger/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func New(
	ctx context.Context,
) (service.Service, func() error, error) {
	db, err := sql.Open("pgx", databaseDSN())
	if err != nil {
		return nil, nil, fmt.Errorf("open PostgreSQL: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = db.PingContext(pingCtx)
	cancel()

	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	client, err := openRedis(ctx)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}

	budgets := pg.NewBudgetRepository(db)
	transactions := pg.NewTransactionRepository(db)
	reports := pg.NewReportRepository(db)
	cache := rediscache.NewReportCache(client)

	svc := internalservice.NewApplication(
		budgets,
		transactions,
		reports,
		cache,
	)

	closeFn := func() error {
		redisErr := client.Close()
		postgresErr := db.Close()

		return errors.Join(redisErr, postgresErr)
	}

	return svc, closeFn, nil
}
