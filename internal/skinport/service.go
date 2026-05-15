package skinport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

const (
	DefaultAppID    = 730
	DefaultCurrency = "EUR"
)

var (
	ErrInvalidAppID        = errors.New("app_id must be positive")
	ErrUnsupportedCurrency = errors.New("unsupported currency")
)

var supportedCurrencies = map[string]struct{}{
	"AUD": {}, "BRL": {}, "CAD": {}, "CHF": {}, "CNY": {}, "CZK": {}, "DKK": {},
	"EUR": {}, "GBP": {}, "HRK": {}, "NOK": {}, "PLN": {}, "RUB": {}, "SEK": {},
	"TRY": {}, "USD": {},
}

type Fetcher interface {
	FetchItems(ctx context.Context, appID int, currency string, tradable bool) ([]Item, error)
}

type Service struct {
	fetcher Fetcher
	ttl     time.Duration
	now     func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
	group singleflight.Group
}

type cacheEntry struct {
	items     []PriceItem
	expiresAt time.Time
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

func NewService(fetcher Fetcher, ttl time.Duration) *Service {
	return &Service{
		fetcher: fetcher,
		ttl:     ttl,
		now:     time.Now,
		cache:   make(map[string]cacheEntry),
	}
}

func (s *Service) Prices(ctx context.Context, appID int, currency string) ([]PriceItem, error) {
	appID, currency, err := NormalizeQuery(appID, currency)
	if err != nil {
		return nil, err
	}

	key := cacheKey(appID, currency)
	if items, ok := s.getCached(key); ok {
		return items, nil
	}

	value, err, _ := s.group.Do(key, func() (any, error) {
		if items, ok := s.getCached(key); ok {
			return items, nil
		}

		items, err := s.fetchAndMerge(ctx, appID, currency)
		if err != nil {
			return nil, err
		}
		s.setCached(key, items)
		return clonePriceItems(items), nil
	})
	if err != nil {
		return nil, err
	}

	items, ok := value.([]PriceItem)
	if !ok {
		return nil, fmt.Errorf("unexpected cache value type")
	}
	return clonePriceItems(items), nil
}

func NormalizeQuery(appID int, currency string) (int, string, error) {
	if appID == 0 {
		appID = DefaultAppID
	}
	if appID < 0 {
		return 0, "", ErrInvalidAppID
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = DefaultCurrency
	}
	if _, ok := supportedCurrencies[currency]; !ok {
		return 0, "", ErrUnsupportedCurrency
	}

	return appID, currency, nil
}

func (s *Service) fetchAndMerge(ctx context.Context, appID int, currency string) ([]PriceItem, error) {
	var tradableItems []Item
	var nonTradableItems []Item

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		items, err := s.fetcher.FetchItems(ctx, appID, currency, true)
		tradableItems = items
		return err
	})
	group.Go(func() error {
		items, err := s.fetcher.FetchItems(ctx, appID, currency, false)
		nonTradableItems = items
		return err
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return mergeItems(tradableItems, nonTradableItems), nil
}

func mergeItems(tradableItems, nonTradableItems []Item) []PriceItem {
	byName := make(map[string]*PriceItem, len(tradableItems)+len(nonTradableItems))

	for _, item := range tradableItems {
		price := priceItemFromSkinport(item)
		price.TradableMinPrice = jsonNumberString(item.MinPrice)
		price.TradableQuantity = item.Quantity
		byName[item.MarketHashName] = &price
	}

	for _, item := range nonTradableItems {
		price, ok := byName[item.MarketHashName]
		if !ok {
			newPrice := priceItemFromSkinport(item)
			price = &newPrice
			byName[item.MarketHashName] = price
		}
		price.NonTradableMinPrice = jsonNumberString(item.MinPrice)
		price.NonTradableQuantity = item.Quantity
	}

	merged := make([]PriceItem, 0, len(byName))
	for _, item := range byName {
		merged = append(merged, *item)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].MarketHashName < merged[j].MarketHashName
	})

	return merged
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

func cacheKey(appID int, currency string) string {
	return fmt.Sprintf("%d:%s", appID, currency)
}

func (s *Service) getCached(key string) ([]PriceItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.cache[key]
	if !ok || !s.now().Before(entry.expiresAt) {
		return nil, false
	}
	return clonePriceItems(entry.items), true
}

func (s *Service) setCached(key string, items []PriceItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[key] = cacheEntry{
		items:     clonePriceItems(items),
		expiresAt: s.now().Add(s.ttl),
	}
}

func clonePriceItems(items []PriceItem) []PriceItem {
	if items == nil {
		return nil
	}
	cloned := make([]PriceItem, len(items))
	copy(cloned, items)
	return cloned
}
