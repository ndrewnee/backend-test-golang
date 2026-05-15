//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
	"github.com/ndrewnee/backend-test-golang/internal/money"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

func TestUsersDebitRouteIntegration(t *testing.T) {
	ctx, pool := openIntegrationDB(t)

	skinportServer, _ := newSkinportIntegrationServer()
	defer skinportServer.Close()

	server := newIntegrationServer(t, pool, skinportServer.URL)

	userID := time.Now().UnixNano()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, balance)
		VALUES ($1, 200.00)
	`, userID)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		server.URL+"/users/"+strconv.FormatInt(userID, 10)+"/debit",
		bytes.NewBufferString(`{"amount":"25.50"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var record dto.DebitResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&record))
	require.Equal(t, userID, record.UserID)
	require.Equal(t, "25.50", record.Amount)
	require.Equal(t, "200.00", record.BalanceBefore)
	require.Equal(t, "174.50", record.BalanceAfter)

	var balance string
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance))
	require.Equal(t, "174.50", balance)

	var historyCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM balance_debits WHERE user_id = $1`, userID).Scan(&historyCount))
	require.Equal(t, 1, historyCount)
}

func TestUsersDebitConcurrentIntegration(t *testing.T) {
	ctx, pool := openIntegrationDB(t)

	const userID = 99
	_, err := pool.Exec(ctx, `
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
