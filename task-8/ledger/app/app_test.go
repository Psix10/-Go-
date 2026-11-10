package app

import (
	"context"
	"errors"
	"testing"
)

func TestNewWithCanceledContext(t *testing.T) {
	t.Setenv(
		"DATABASE_URL",
		"postgres://postgres:postgres@127.0.0.1:5444/cashapp?sslmode=disable",
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc, closeFn, err := New(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}

	if svc != nil {
		t.Error("service must be nil after initialization failure")
	}

	if closeFn != nil {
		t.Error("close function must be nil after initialization failure")
	}
}
