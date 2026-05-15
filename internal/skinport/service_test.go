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
)

func TestServicePricesMergesAndCachesSkinportItems(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		if got := r.Header.Get("Accept-Encoding"); got != "br" {
			t.Errorf("Accept-Encoding = %q, want br", got)
		}
		if got := r.URL.Query().Get("app_id"); got != "730" {
			t.Errorf("app_id = %q, want 730", got)
		}
		if got := r.URL.Query().Get("currency"); got != "USD" {
			t.Errorf("currency = %q, want USD", got)
		}

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
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	service := NewService(client, time.Minute)

	first, err := service.Prices(context.Background(), 730, "usd")
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}
	second, err := service.Prices(context.Background(), 730, "USD")
	if err != nil {
		t.Fatalf("Prices() cached error = %v", err)
	}

	if calls.Load() != 2 {
		t.Fatalf("Skinport calls = %d, want 2", calls.Load())
	}
	if len(first) != 2 {
		t.Fatalf("len(first) = %d, want 2", len(first))
	}
	if len(second) != 2 {
		t.Fatalf("len(second) = %d, want 2", len(second))
	}

	ak := first[0]
	if ak.MarketHashName != "AK-47" {
		t.Fatalf("first item = %s, want AK-47", ak.MarketHashName)
	}
	if stringValue(ak.TradableMinPrice) != "10.25" {
		t.Fatalf("tradable min = %s, want 10.25", stringValue(ak.TradableMinPrice))
	}
	if stringValue(ak.NonTradableMinPrice) != "9.99" {
		t.Fatalf("non-tradable min = %s, want 9.99", stringValue(ak.NonTradableMinPrice))
	}
	if first[1].TradableMinPrice != nil {
		t.Fatalf("expected nil tradable min for second item, got %s", stringValue(first[1].TradableMinPrice))
	}
}

func TestNormalizeQuery(t *testing.T) {
	t.Parallel()

	appID, currency, err := NormalizeQuery(0, "")
	if err != nil {
		t.Fatalf("NormalizeQuery() error = %v", err)
	}
	if appID != DefaultAppID || currency != DefaultCurrency {
		t.Fatalf("NormalizeQuery() = %d, %s", appID, currency)
	}

	if _, _, err := NormalizeQuery(-1, "USD"); err == nil {
		t.Fatal("expected error for negative app_id")
	}
	if _, _, err := NormalizeQuery(730, "XXX"); err == nil {
		t.Fatal("expected error for unsupported currency")
	}
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
			t.Fatal(err)
		}

		writer := newBrotliHTTPWriter(w)
		if _, err := writer.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		if got := r.URL.Query().Get("tradable"); got != "1" {
			t.Errorf("tradable = %q, want 1", got)
		}
		if _, err := strconv.Atoi(r.URL.Query().Get("app_id")); err != nil {
			t.Errorf("app_id is not integer: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	items, err := client.FetchItems(context.Background(), 730, "EUR", true)
	if err != nil {
		t.Fatalf("FetchItems() error = %v", err)
	}
	if len(items) != 1 || items[0].MarketHashName != "Item" {
		t.Fatalf("items = %#v", items)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
