package prices

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
)

func TestHandlerItemsPrices(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubPriceService{
		items: []dto.PriceItem{{
			MarketHashName: "AK-47",
			Currency:       "EUR",
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/items/prices?app_id=730&currency=EUR", nil)
	res := httptest.NewRecorder()

	handler.ItemsPrices(res, req)

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	var response dto.ItemsPricesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&response))
	require.Len(t, response.Items, 1)
	require.Equal(t, "AK-47", response.Items[0].MarketHashName)
}

func TestHandlerItemsPricesValidation(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubPriceService{})
	req := httptest.NewRequest(http.MethodGet, "/items/prices?app_id=abc", nil)
	res := httptest.NewRecorder()

	handler.ItemsPrices(res, req)

	require.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())
}

type stubPriceService struct {
	items []dto.PriceItem
	err   error
}

func (s stubPriceService) Prices(_ context.Context, _ int, _ string) ([]dto.PriceItem, error) {
	return s.items, s.err
}
