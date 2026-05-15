package users

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
)

func TestHandlerDebitUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "success", path: "/users/1/debit", body: `{"amount":"100.00"}`, wantStatus: http.StatusOK},
		{name: "invalid amount", path: "/users/1/debit", body: `{"amount":"1.001"}`, serviceErr: newInvalidAmountError(errors.New("amount must have at most 2 decimal places")), wantStatus: http.StatusBadRequest},
		{name: "unknown user", path: "/users/2/debit", body: `{"amount":"100.00"}`, serviceErr: ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "insufficient funds", path: "/users/1/debit", body: `{"amount":"9999.00"}`, serviceErr: ErrInsufficientFunds, wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := http.NewServeMux()
			router.HandleFunc("POST /users/{id}/debit", NewHandler(&stubUserService{err: tt.serviceErr}).DebitUser)
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body))
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			require.Equal(t, tt.wantStatus, res.Code, res.Body.String())
		})
	}
}

type stubUserService struct {
	err error
}

func (s *stubUserService) Debit(_ context.Context, userID int64, request dto.DebitRequest) (dto.DebitResponse, error) {
	if s.err != nil {
		return dto.DebitResponse{}, s.err
	}
	if request.Amount == "" {
		return dto.DebitResponse{}, errors.New("amount should be decoded before service call")
	}
	return dto.DebitResponse{
		ID:            1,
		UserID:        userID,
		Amount:        request.Amount,
		BalanceBefore: "1000.00",
		BalanceAfter:  "900.00",
		CreatedAt:     time.Unix(1, 0).UTC(),
	}, nil
}
