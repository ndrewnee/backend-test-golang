package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/ndrewnee/backend-test-golang/internal/models"
	"github.com/ndrewnee/backend-test-golang/internal/money"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Debit(ctx context.Context, userID int64, amount decimal.Decimal) (models.BalanceDebit, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return models.BalanceDebit{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	user := models.User{ID: userID}
	err = tx.QueryRow(ctx, `
		SELECT balance::text
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(&user.Balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.BalanceDebit{}, ErrUserNotFound
	}
	if err != nil {
		return models.BalanceDebit{}, err
	}

	balance, err := money.ParseDatabaseValue(user.Balance)
	if err != nil {
		return models.BalanceDebit{}, err
	}
	if balance.LessThan(amount) {
		return models.BalanceDebit{}, ErrInsufficientFunds
	}

	after := balance.Sub(amount)
	amountRaw := money.Format(amount)
	beforeRaw := money.Format(balance)
	afterRaw := money.Format(after)

	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET balance = $2::numeric(18,2)
		WHERE id = $1
	`, userID, afterRaw); err != nil {
		return models.BalanceDebit{}, fmt.Errorf("update user balance: %w", err)
	}

	record := models.BalanceDebit{
		UserID:        userID,
		Amount:        amountRaw,
		BalanceBefore: beforeRaw,
		BalanceAfter:  afterRaw,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO balance_debits (user_id, amount, balance_before, balance_after)
		VALUES ($1, $2::numeric(18,2), $3::numeric(18,2), $4::numeric(18,2))
		RETURNING id, created_at
	`, userID, amountRaw, beforeRaw, afterRaw).Scan(&record.ID, &record.CreatedAt)
	if err != nil {
		return models.BalanceDebit{}, fmt.Errorf("insert debit history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.BalanceDebit{}, err
	}

	return record, nil
}
