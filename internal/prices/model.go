package prices

import "encoding/json"

type Item struct {
	MarketHashName string       `json:"market_hash_name"`
	Currency       string       `json:"currency"`
	SuggestedPrice *json.Number `json:"suggested_price"`
	ItemPage       string       `json:"item_page"`
	MarketPage     string       `json:"market_page"`
	MinPrice       *json.Number `json:"min_price"`
	MaxPrice       *json.Number `json:"max_price"`
	MeanPrice      *json.Number `json:"mean_price"`
	MedianPrice    *json.Number `json:"median_price"`
	Quantity       int          `json:"quantity"`
	CreatedAt      int64        `json:"created_at"`
	UpdatedAt      int64        `json:"updated_at"`
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

func priceItemFromSkinport(item Item) PriceItem {
	return PriceItem{
		MarketHashName:    item.MarketHashName,
		Currency:          item.Currency,
		SuggestedPrice:    jsonNumberString(item.SuggestedPrice),
		ItemPage:          item.ItemPage,
		MarketPage:        item.MarketPage,
		Quantity:          item.Quantity,
		SkinportCreatedAt: item.CreatedAt,
		SkinportUpdatedAt: item.UpdatedAt,
	}
}

func jsonNumberString(number *json.Number) *string {
	if number == nil {
		return nil
	}
	value := number.String()
	return &value
}
