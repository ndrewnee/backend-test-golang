package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ndrewnee/backend-test-golang/internal/skinport"
	"github.com/ndrewnee/backend-test-golang/internal/users"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestDebitUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "success", path: "/users/1/debit", body: `{"amount":"100.00"}`, wantStatus: http.StatusOK},
		{name: "invalid amount", path: "/users/1/debit", body: `{"amount":"1.001"}`, wantStatus: http.StatusBadRequest},
		{name: "unknown user", path: "/users/2/debit", body: `{"amount":"100.00"}`, serviceErr: users.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "insufficient funds", path: "/users/1/debit", body: `{"amount":"9999.00"}`, serviceErr: users.ErrInsufficientFunds, wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := NewRouter(stubPriceService{}, &stubUserService{err: tt.serviceErr})
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body))
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			require.Equal(t, tt.wantStatus, res.Code, res.Body.String())
		})
	}
}

func TestItemsPrices(t *testing.T) {
	t.Parallel()

	router := NewRouter(stubPriceService{
		items: []skinport.PriceItem{{
			MarketHashName: "AK-47",
			Currency:       "EUR",
		}},
	}, &stubUserService{})

	req := httptest.NewRequest(http.MethodGet, "/items/prices?app_id=730&currency=EUR", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	var response struct {
		Items []skinport.PriceItem `json:"items"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&response))
	require.Len(t, response.Items, 1)
	require.Equal(t, "AK-47", response.Items[0].MarketHashName)
}

func TestItemsPricesValidation(t *testing.T) {
	t.Parallel()

	router := NewRouter(stubPriceService{}, &stubUserService{})

	req := httptest.NewRequest(http.MethodGet, "/items/prices?app_id=abc", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())
}

type stubPriceService struct {
	items []skinport.PriceItem
	err   error
}

func (s stubPriceService) Prices(_ context.Context, _ int, _ string) ([]skinport.PriceItem, error) {
	return s.items, s.err
}

type stubUserService struct {
	err error
}

func (s *stubUserService) Debit(_ context.Context, userID int64, amount decimal.Decimal) (users.DebitRecord, error) {
	if s.err != nil {
		return users.DebitRecord{}, s.err
	}
	if amount.IsZero() {
		return users.DebitRecord{}, errors.New("amount should be validated before service call")
	}
	return users.DebitRecord{
		ID:            1,
		UserID:        userID,
		Amount:        amount.StringFixed(2),
		BalanceBefore: "1000.00",
		BalanceAfter:  "900.00",
		CreatedAt:     time.Unix(1, 0).UTC(),
	}, nil
}
