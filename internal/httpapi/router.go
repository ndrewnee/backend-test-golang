package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
	"github.com/ndrewnee/backend-test-golang/internal/prices"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

func NewRouter(priceHandler *prices.Handler, userHandler *users.Handler) http.Handler {
	router := chi.NewRouter()
	router.Get("/healthz", healthz)
	router.Get("/items/prices", priceHandler.ItemsPrices)
	router.Post("/users/{id}/debit", userHandler.DebitUser)

	return router
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, dto.HealthResponse{Status: "ok"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
