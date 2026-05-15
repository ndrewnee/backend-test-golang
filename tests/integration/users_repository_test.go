//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/db"
	"github.com/ndrewnee/backend-test-golang/internal/money"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

func TestDebitConcurrentIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, db.RunMigrations(ctx, pool))

	const userID = 99
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, balance)
		VALUES ($1, 100.00)
		ON CONFLICT (id) DO UPDATE SET balance = EXCLUDED.balance
	`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM balance_debits WHERE user_id = $1`, userID)
	require.NoError(t, err)

	amount, err := money.ParseAmount("60.00")
	require.NoError(t, err)

	repository := users.NewRepository(pool)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repository.Debit(ctx, userID, amount)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	var successCount int
	var insufficientCount int
	for err := range errs {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, users.ErrInsufficientFunds):
			insufficientCount++
		default:
			require.NoError(t, err)
		}
	}

	require.Equal(t, 1, successCount)
	require.Equal(t, 1, insufficientCount)

	var balance string
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance))
	require.Equal(t, "40.00", balance)
}
