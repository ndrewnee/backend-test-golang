package skinport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

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

func NewClient(rawBaseURL string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse skinport base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("skinport base url must be absolute")
	}

	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) FetchItems(ctx context.Context, appID int, currency string, tradable bool) ([]Item, error) {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/items"

	query := endpoint.Query()
	query.Set("app_id", strconv.Itoa(appID))
	query.Set("currency", strings.ToUpper(currency))
	if tradable {
		query.Set("tradable", "1")
	} else {
		query.Set("tradable", "0")
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "br")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch skinport items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("skinport returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	reader := io.Reader(resp.Body)
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "br") {
		reader = brotli.NewReader(resp.Body)
	}

	decoder := json.NewDecoder(reader)
	decoder.UseNumber()

	var items []Item
	if err := decoder.Decode(&items); err != nil {
		return nil, fmt.Errorf("decode skinport response: %w", err)
	}

	return items, nil
}
