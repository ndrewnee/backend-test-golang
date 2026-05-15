package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
	"github.com/ndrewnee/backend-test-golang/internal/models"
)

func TestServiceDebit(t *testing.T) {
	t.Parallel()

	createdAt := time.Unix(1, 0).UTC()
	service := NewService(&stubDebitRepository{
		record: models.BalanceTransaction{
			ID:            1,
			UserID:        42,
			Amount:        "25.50",
			BalanceBefore: "200.00",
			BalanceAfter:  "174.50",
			CreatedAt:     createdAt,
		},
	})

	got, err := service.Debit(context.Background(), 42, dto.DebitRequest{Amount: "25.50"})

	require.NoError(t, err)
	require.Equal(t, dto.DebitResponse{
		ID:            1,
		UserID:        42,
		Amount:        "25.50",
		BalanceBefore: "200.00",
		BalanceAfter:  "174.50",
		CreatedAt:     createdAt,
	}, got)
}

func TestServiceDebitValidation(t *testing.T) {
	t.Parallel()

	repository := &stubDebitRepository{}
	service := NewService(repository)

	_, err := service.Debit(context.Background(), 42, dto.DebitRequest{Amount: "1.001"})

	require.ErrorIs(t, err, ErrInvalidAmount)
	require.False(t, repository.called)
}

func TestServiceDebitRepositoryErrors(t *testing.T) {
	t.Parallel()

	tests := []error{
		ErrUserNotFound,
		ErrInsufficientFunds,
		errors.New("database unavailable"),
	}

	for _, wantErr := range tests {
		t.Run(wantErr.Error(), func(t *testing.T) {
			t.Parallel()

			service := NewService(&stubDebitRepository{err: wantErr})

			_, err := service.Debit(context.Background(), 42, dto.DebitRequest{Amount: "10.00"})

			require.ErrorIs(t, err, wantErr)
		})
	}
}

type stubDebitRepository struct {
	record models.BalanceTransaction
	err    error
	called bool
}

func (r *stubDebitRepository) Debit(_ context.Context, _ int64, _ decimal.Decimal) (models.BalanceTransaction, error) {
	r.called = true
	if r.err != nil {
		return models.BalanceTransaction{}, r.err
	}
	return r.record, nil
}
