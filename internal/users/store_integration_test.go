package users

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ndrewnee/backend-test-golang/internal/db"
	"github.com/ndrewnee/backend-test-golang/internal/money"
)

func TestDebitConcurrentIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	const userID = 99
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, balance)
		VALUES ($1, 100.00)
		ON CONFLICT (id) DO UPDATE SET balance = EXCLUDED.balance
	`, userID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM balance_debits WHERE user_id = $1`, userID)
	if err != nil {
		t.Fatalf("clean history: %v", err)
	}

	amount, err := money.ParseAmount("60.00")
	if err != nil {
		t.Fatalf("parse amount: %v", err)
	}

	store := NewStore(pool)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Debit(ctx, userID, amount)
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
		case errors.Is(err, ErrInsufficientFunds):
			insufficientCount++
		default:
			t.Fatalf("unexpected debit error: %v", err)
		}
	}

	if successCount != 1 || insufficientCount != 1 {
		t.Fatalf("success=%d insufficient=%d, want 1 and 1", successCount, insufficientCount)
	}

	var balance string
	if err := pool.QueryRow(ctx, `SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance); err != nil {
		t.Fatalf("read balance: %v", err)
	}
	if balance != "40.00" {
		t.Fatalf("balance = %s, want 40.00", balance)
	}
}
