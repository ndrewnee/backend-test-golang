package skinport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicePricesMergesAndCachesSkinportItems(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		assert.Equal(t, "br", r.Header.Get("Accept-Encoding"))
		assert.Equal(t, "730", r.URL.Query().Get("app_id"))
		assert.Equal(t, "USD", r.URL.Query().Get("currency"))

		w.Header().Set("Content-Type", "application/json")
		tradable := r.URL.Query().Get("tradable")
		if tradable == "1" {
			_, _ = w.Write([]byte(`[
				{"market_hash_name":"AK-47","currency":"USD","suggested_price":12.5,"item_page":"https://example.test/i","market_page":"https://example.test/m","min_price":10.25,"quantity":3,"created_at":1,"updated_at":2},
				{"market_hash_name":"M4A1","currency":"USD","suggested_price":7,"item_page":"https://example.test/i2","market_page":"https://example.test/m2","min_price":null,"quantity":1,"created_at":3,"updated_at":4}
			]`))
			return
		}

		if tradable == "0" {
			_, _ = w.Write([]byte(`[
				{"market_hash_name":"AK-47","currency":"USD","suggested_price":12.5,"item_page":"https://example.test/i","market_page":"https://example.test/m","min_price":9.99,"quantity":2,"created_at":1,"updated_at":5}
			]`))
			return
		}

		http.Error(w, "missing tradable", http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	require.NoError(t, err)
	service := NewService(client, time.Minute)

	first, err := service.Prices(context.Background(), 730, "usd")
	require.NoError(t, err)
	second, err := service.Prices(context.Background(), 730, "USD")
	require.NoError(t, err)

	require.Equal(t, int64(2), calls.Load())
	require.Len(t, first, 2)
	require.Len(t, second, 2)

	ak := first[0]
	require.Equal(t, "AK-47", ak.MarketHashName)
	require.Equal(t, "10.25", stringValue(ak.TradableMinPrice))
	require.Equal(t, "9.99", stringValue(ak.NonTradableMinPrice))
	require.Nil(t, first[1].TradableMinPrice)
}

func TestNormalizeQuery(t *testing.T) {
	t.Parallel()

	appID, currency, err := NormalizeQuery(0, "")
	require.NoError(t, err)
	require.Equal(t, DefaultAppID, appID)
	require.Equal(t, DefaultCurrency, currency)

	_, _, err = NormalizeQuery(-1, "USD")
	require.Error(t, err)

	_, _, err = NormalizeQuery(730, "XXX")
	require.Error(t, err)
}

func TestClientDecodesBrotliResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Content-Type", "application/json")

		payload, err := json.Marshal([]map[string]any{{
			"market_hash_name": "Item",
			"currency":         "EUR",
			"min_price":        1.25,
			"quantity":         1,
		}})
		if err != nil {
			assert.NoError(t, err)
			return
		}

		writer := newBrotliHTTPWriter(w)
		if _, err := writer.Write(payload); err != nil {
			assert.NoError(t, err)
			return
		}
		if err := writer.Close(); err != nil {
			assert.NoError(t, err)
			return
		}

		assert.Equal(t, "1", r.URL.Query().Get("tradable"))
		if _, err := strconv.Atoi(r.URL.Query().Get("app_id")); err != nil {
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	require.NoError(t, err)

	items, err := client.FetchItems(context.Background(), 730, "EUR", true)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Item", items[0].MarketHashName)
}

func stringValue(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
