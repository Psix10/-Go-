package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"gateway/internal/api"

	"ledger/app"
	"ledger/service"
)

const serverAddress = ":8080"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	svc, closeFn, err := app.New(ctx)
	if err != nil {
		return fmt.Errorf("Ledger initialization failed: %w", err)
	}

	defer func() {
		if err := closeFn(); err != nil {
			log.Printf("Ledger resources close failed: %v", err)
		}
	}()

	server := &http.Server{
		Addr:              serverAddress,
		Handler:           newGatewayHandler(svc),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	log.Printf("Gateway started on http://localhost%s", serverAddress)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("Gateway HTTP server failed: %w", err)
		}
		return nil

	case <-ctx.Done():
		stop()
		log.Println("Gateway shutdown started")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("Gateway shutdown failed: %w", err)
	}

	if err := <-serverErrors; err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("Gateway HTTP server failed: %w", err)
	}

	return nil
}

func newGatewayHandler(svc service.Service) http.Handler {
	routes := api.New(svc).Routes()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		routes.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}
