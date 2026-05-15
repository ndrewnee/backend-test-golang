package users

import "time"

type DebitRecord struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Amount        string    `json:"amount"`
	BalanceBefore string    `json:"balance_before"`
	BalanceAfter  string    `json:"balance_after"`
	CreatedAt     time.Time `json:"created_at"`
}
