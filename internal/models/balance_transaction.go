package models

import "time"

type BalanceTransaction struct {
	ID            int64
	UserID        int64
	Amount        string
	BalanceBefore string
	BalanceAfter  string
	CreatedAt     time.Time
}
