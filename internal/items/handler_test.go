package items

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
)

func TestHandlerItems(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubItemsService{
		items: []dto.Item{{
			MarketHashName: "AK-47",
			Currency:       "EUR",
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/items?app_id=730&currency=EUR", nil)
	res := httptest.NewRecorder()

	handler.Items(res, req)

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	var response dto.ItemsResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&response))
	require.Len(t, response.Items, 1)
	require.Equal(t, "AK-47", response.Items[0].MarketHashName)
}

func TestHandlerItemsValidation(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubItemsService{})
	req := httptest.NewRequest(http.MethodGet, "/items?app_id=abc", nil)
	res := httptest.NewRecorder()

	handler.Items(res, req)

	require.Equal(t, http.StatusBadRequest, res.Code, res.Body.String())
}

type stubItemsService struct {
	items []dto.Item
	err   error
}

func (s stubItemsService) Items(_ context.Context, _ int, _ string) ([]dto.Item, error) {
	return s.items, s.err
}
