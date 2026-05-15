package items

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
)

type ItemsService interface {
	Items(ctx context.Context, appID int, currency string) ([]dto.Item, error)
}

type Handler struct {
	service ItemsService
}

func NewHandler(service ItemsService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Items(w http.ResponseWriter, r *http.Request) {
	appID := 0
	if raw := r.URL.Query().Get("app_id"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "app_id must be a positive integer")
			return
		}
		appID = parsed
	}

	items, err := h.service.Items(r.Context(), appID, r.URL.Query().Get("currency"))
	if err != nil {
		if errors.Is(err, ErrUnsupportedCurrency) || errors.Is(err, ErrInvalidAppID) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, r.Context().Err()) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Warn("failed to fetch items", "error", err)
		writeError(w, http.StatusBadGateway, "failed to fetch items")
		return
	}

	writeJSON(w, http.StatusOK, dto.ItemsResponse{Items: items})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}
