package rajaongkir

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/config"
)

type Config struct {
	APIKey   string
	BaseURL  string
	OriginID int
}

type Client struct {
	apiKey     string
	baseURL    string
	originID   int
	httpClient *http.Client
}

func NewClient(cfg config.RajaOngkirConfig) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://rajaongkir.komerce.id/api/v1"
	}

	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		originID:   cfg.OriginID,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SearchDestination(ctx context.Context, keyword string, limit, offset int) ([]DomesticDestination, error) {
	if limit <= 0 {
		limit = 10
	}

	endpoint := fmt.Sprintf("%s/destination/domestic-destination?search=%s&limit=%d&offset=%d",
		c.baseURL,
		url.QueryEscape(keyword),
		limit,
		offset,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var result BaseResponse[[]DomesticDestination]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Meta.Code != http.StatusOK && result.Meta.Code != 0 {
		return nil, errors.New(result.Meta.Message)
	}

	return result.Data, nil
}

func (c *Client) CalculateCost(ctx context.Context, reqData CalculateDomesticCostRequest) ([]ShippingCostResult, error) {
	endpoint := fmt.Sprintf("%s/calculate/domestic-cost", c.baseURL)

	if reqData.WeightGrams <= 0 {
		reqData.WeightGrams = 1000
	}

	originID := reqData.OriginID
	if originID == 0 {
		originID = c.originID
	}

	courier := reqData.Courier
	if courier == "" {
		courier = "jne:pos:tiki:sicepat:jnt"
	}

	price := reqData.Price
	if price == "" {
		price = "lowest"
	}

	form := url.Values{}
	form.Set("origin", strconv.Itoa(originID))
	form.Set("destination", strconv.Itoa(reqData.DestinationID))
	form.Set("weight", strconv.Itoa(reqData.WeightGrams))
	form.Set("courier", courier)
	form.Set("price", price)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("key", c.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cost: %w", err)
	}
	defer resp.Body.Close()

	var result BaseResponse[[]ShippingCostResult]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Meta.Code != http.StatusOK && result.Meta.Code != 0 {
		return nil, errors.New(result.Meta.Message)
	}

	return result.Data, nil
}
