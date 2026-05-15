//go:build integration

package integration_test

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/db"
	"github.com/ndrewnee/backend-test-golang/internal/httpapi"
	"github.com/ndrewnee/backend-test-golang/internal/prices"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

func openIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, db.RunMigrations(ctx, pool))

	return ctx, pool
}

func newIntegrationServer(t *testing.T, pool *pgxpool.Pool, skinportBaseURL string) *httptest.Server {
	t.Helper()

	skinportClient, err := prices.NewClient(skinportBaseURL, time.Second)
	require.NoError(t, err)

	priceService := prices.NewService(skinportClient, time.Minute)
	userRepository := users.NewRepository(pool)
	userService := users.NewService(userRepository)

	server := httptest.NewServer(httpapi.NewRouter(
		prices.NewHandler(priceService),
		users.NewHandler(userService),
	))
	t.Cleanup(server.Close)

	return server
}
