package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndrewnee/backend-test-golang/internal/config"
	"github.com/ndrewnee/backend-test-golang/internal/db"
	"github.com/ndrewnee/backend-test-golang/internal/httpapi"
	"github.com/ndrewnee/backend-test-golang/internal/skinport"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	if cfg.RunMigrations {
		if err := db.RunMigrations(ctx, pool); err != nil {
			return err
		}
	}

	skinportClient, err := skinport.NewClient(cfg.SkinportBaseURL, cfg.SkinportTimeout)
	if err != nil {
		return err
	}

	priceService := skinport.NewService(skinportClient, cfg.SkinportCacheTTL)
	userStore := users.NewStore(pool)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(priceService, userStore),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
