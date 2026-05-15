package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/ndrewnee/backend-test-golang/internal/prices"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

type PriceService interface {
	Prices(ctx context.Context, appID int, currency string) ([]prices.PriceItem, error)
}

type UserService interface {
	Debit(ctx context.Context, userID int64, amount decimal.Decimal) (users.DebitRecord, error)
}

func NewRouter(priceService PriceService, userService UserService) http.Handler {
	handler := &Handler{
		priceService: priceService,
		userService:  userService,
	}

	router := chi.NewRouter()
	router.Get("/healthz", handler.healthz)
	router.Get("/items/prices", handler.itemsPrices)
	router.Post("/users/{id}/debit", handler.debitUser)

	return router
}
