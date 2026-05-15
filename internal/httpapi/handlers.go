package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ndrewnee/backend-test-golang/internal/money"
	"github.com/ndrewnee/backend-test-golang/internal/skinport"
	"github.com/ndrewnee/backend-test-golang/internal/users"
)

type Handler struct {
	priceService PriceService
	userService  UserService
}

type errorResponse struct {
	Error string `json:"error"`
}

type debitRequest struct {
	Amount string `json:"amount"`
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) itemsPrices(w http.ResponseWriter, r *http.Request) {
	appID := 0
	if raw := r.URL.Query().Get("app_id"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "app_id must be a positive integer")
			return
		}
		appID = parsed
	}

	items, err := h.priceService.Prices(r.Context(), appID, r.URL.Query().Get("currency"))
	if err != nil {
		if errors.Is(err, skinport.ErrUnsupportedCurrency) || errors.Is(err, skinport.ErrInvalidAppID) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, r.Context().Err()) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Warn("failed to fetch item prices", "error", err)
		writeError(w, http.StatusBadGateway, "failed to fetch item prices")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) debitUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || userID <= 0 {
		writeError(w, http.StatusBadRequest, "user id must be a positive integer")
		return
	}

	var request debitRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	amount, err := money.ParseAmount(request.Amount)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	record, err := h.userService.Debit(r.Context(), userID, amount)
	if errors.Is(err, users.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if errors.Is(err, users.ErrInsufficientFunds) {
		writeError(w, http.StatusConflict, "insufficient funds")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to debit user balance")
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
