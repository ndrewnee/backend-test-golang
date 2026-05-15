package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
)

type debitService interface {
	Debit(ctx context.Context, userID int64, request dto.DebitRequest) (dto.DebitResponse, error)
}

type Handler struct {
	service debitService
}

func NewHandler(service debitService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) DebitUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || userID <= 0 {
		writeError(w, http.StatusBadRequest, "user id must be a positive integer")
		return
	}

	var request dto.DebitRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	record, err := h.service.Debit(r.Context(), userID, request)
	if errors.Is(err, ErrInvalidAmount) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if errors.Is(err, ErrInsufficientFunds) {
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
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}
