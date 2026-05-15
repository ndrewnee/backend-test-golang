package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/ndrewnee/backend-test-golang/internal/money"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Store struct {
	pool *pgxpool.Pool
}

type DebitRecord struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Amount        string    `json:"amount"`
	BalanceBefore string    `json:"balance_before"`
	BalanceAfter  string    `json:"balance_after"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Debit(ctx context.Context, userID int64, amount decimal.Decimal) (DebitRecord, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return DebitRecord{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var balanceRaw string
	err = tx.QueryRow(ctx, `
		SELECT balance::text
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(&balanceRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return DebitRecord{}, ErrUserNotFound
	}
	if err != nil {
		return DebitRecord{}, err
	}

	balance, err := money.ParseDatabaseValue(balanceRaw)
	if err != nil {
		return DebitRecord{}, err
	}
	if balance.LessThan(amount) {
		return DebitRecord{}, ErrInsufficientFunds
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
		return DebitRecord{}, fmt.Errorf("update user balance: %w", err)
	}

	record := DebitRecord{
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
		return DebitRecord{}, fmt.Errorf("insert debit history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return DebitRecord{}, err
	}

	return record, nil
}
