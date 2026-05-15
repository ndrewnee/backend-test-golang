//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
)

func TestItemsIntegration(t *testing.T) {
	_, pool := openIntegrationDB(t)

	skinportServer, skinportCalls := newSkinportIntegrationServer()
	defer skinportServer.Close()

	server := newIntegrationServer(t, pool, skinportServer.URL)

	resp, err := server.Client().Get(server.URL + "/items?app_id=730&currency=USD")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body dto.ItemsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Len(t, body.Items, 1)
	require.Equal(t, "AK-47", body.Items[0].MarketHashName)
	require.Equal(t, "10.25", stringValue(body.Items[0].TradableMinPrice))
	require.Equal(t, "9.99", stringValue(body.Items[0].NonTradableMinPrice))
	require.Equal(t, int64(2), skinportCalls.Load())
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
