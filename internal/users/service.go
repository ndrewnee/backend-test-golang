package users

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/ndrewnee/backend-test-golang/internal/dto"
	"github.com/ndrewnee/backend-test-golang/internal/models"
	"github.com/ndrewnee/backend-test-golang/internal/money"
)

type UserRepository interface {
	Debit(ctx context.Context, userID int64, amount decimal.Decimal) (models.BalanceDebit, error)
}

type Service struct {
	repo UserRepository
}

func NewService(repo UserRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Debit(ctx context.Context, userID int64, request dto.DebitRequest) (dto.DebitResponse, error) {
	amount, err := money.ParseAmount(request.Amount)
	if err != nil {
		return dto.DebitResponse{}, newInvalidAmountError(err)
	}

	record, err := s.repo.Debit(ctx, userID, amount)
	if err != nil {
		return dto.DebitResponse{}, err
	}

	return debitResponseFromModel(record), nil
}

func debitResponseFromModel(record models.BalanceDebit) dto.DebitResponse {
	return dto.DebitResponse{
		ID:            record.ID,
		UserID:        record.UserID,
		Amount:        record.Amount,
		BalanceBefore: record.BalanceBefore,
		BalanceAfter:  record.BalanceAfter,
		CreatedAt:     record.CreatedAt,
	}
}
