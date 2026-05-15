package dto

import "time"

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type DebitRequest struct {
	Amount string `json:"amount"`
}

type DebitResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Amount        string    `json:"amount"`
	BalanceBefore string    `json:"balance_before"`
	BalanceAfter  string    `json:"balance_after"`
	CreatedAt     time.Time `json:"created_at"`
}

type ItemsPricesResponse struct {
	Items []PriceItem `json:"items"`
}

type PriceItem struct {
	MarketHashName      string  `json:"market_hash_name"`
	Currency            string  `json:"currency"`
	SuggestedPrice      *string `json:"suggested_price"`
	ItemPage            string  `json:"item_page"`
	MarketPage          string  `json:"market_page"`
	Quantity            int     `json:"quantity"`
	TradableMinPrice    *string `json:"tradable_min_price"`
	NonTradableMinPrice *string `json:"non_tradable_min_price"`
	TradableQuantity    int     `json:"tradable_quantity"`
	NonTradableQuantity int     `json:"non_tradable_quantity"`
	SkinportCreatedAt   int64   `json:"skinport_created_at"`
	SkinportUpdatedAt   int64   `json:"skinport_updated_at"`
}
