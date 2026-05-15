//go:build integration

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/db"
	"github.com/ndrewnee/backend-test-golang/internal/httpapi"
	"github.com/ndrewnee/backend-test-golang/internal/skinport"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

func TestRoutesIntegration(t *testing.T) {
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

	skinportServer, skinportCalls := newSkinportIntegrationServer()
	defer skinportServer.Close()

	skinportClient, err := skinport.NewClient(skinportServer.URL, time.Second)
	require.NoError(t, err)

	router := httpapi.NewRouter(
		skinport.NewService(skinportClient, time.Minute),
		users.NewStore(pool),
	)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("items prices route fetches and merges Skinport prices", func(t *testing.T) {
		resp, err := server.Client().Get(server.URL + "/items/prices?app_id=730&currency=USD")
		require.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Items []skinport.PriceItem `json:"items"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body.Items, 1)
		require.Equal(t, "AK-47", body.Items[0].MarketHashName)
		require.Equal(t, "10.25", stringValue(body.Items[0].TradableMinPrice))
		require.Equal(t, "9.99", stringValue(body.Items[0].NonTradableMinPrice))
		require.Equal(t, int64(2), skinportCalls.Load())
	})

	t.Run("debit route updates balance and writes history", func(t *testing.T) {
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

		var record users.DebitRecord
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
	})
}

func newSkinportIntegrationServer() (*httptest.Server, *atomic.Int64) {
	var calls atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		if r.Header.Get("Accept-Encoding") != "br" {
			http.Error(w, "missing Brotli encoding request", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("app_id") != "730" || r.URL.Query().Get("currency") != "USD" {
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("tradable") {
		case "1":
			_, _ = w.Write([]byte(`[
				{"market_hash_name":"AK-47","currency":"USD","suggested_price":12.5,"item_page":"https://example.test/i","market_page":"https://example.test/m","min_price":10.25,"quantity":3,"created_at":1,"updated_at":2}
			]`))
		case "0":
			_, _ = w.Write([]byte(`[
				{"market_hash_name":"AK-47","currency":"USD","suggested_price":12.5,"item_page":"https://example.test/i","market_page":"https://example.test/m","min_price":9.99,"quantity":2,"created_at":1,"updated_at":5}
			]`))
		default:
			http.Error(w, "missing tradable", http.StatusBadRequest)
		}
	}))

	return server, &calls
}

func stringValue(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
