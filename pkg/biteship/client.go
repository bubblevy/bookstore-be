package biteship

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client interface {
	SearchArea(ctx context.Context, keyword string) ([]AreaItem, error)
	GetCouriers(ctx context.Context) ([]CourierDetail, error)
	CalculateRatesByAreaID(ctx context.Context, destAreaID, couriers string, items []RateItem) ([]ShippingRatePricing, error)
	CreateOrderByAreaID(ctx context.Context, req CreateOrderByAreaIDRequest) (*OrderResponse, error)
	GetOrderByID(ctx context.Context, orderID string) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string, reason string) (*CancelOrderResponse, error)
	TrackByID(ctx context.Context, trackingID string) (*TrackingDetailResult, error)
	TrackWaybill(ctx context.Context, waybillID, courierCode string) (*TrackingDetailResult, error)
}

type rateCacheEntry struct {
	pricing   []ShippingRatePricing
	expiresAt time.Time
}

type client struct {
	apiKey       string
	baseURL      string
	originAreaID string
	httpClient   *http.Client
	rateMu       sync.RWMutex
	rateCache    map[string]rateCacheEntry
}

func NewClient(apiKey, baseURL, originAreaID string) Client {
	endpoint := strings.TrimRight(baseURL, "/")
	if endpoint == "" {
		endpoint = "https://api.biteship.com"
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &client{
		apiKey:       apiKey,
		baseURL:      endpoint,
		originAreaID: originAreaID,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
		},
		rateCache: make(map[string]rateCacheEntry),
	}
}

func (c *client) doRequest(ctx context.Context, method, endpoint string, body interface{}, target interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errObj struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if jsonErr := json.Unmarshal(respBytes, &errObj); jsonErr == nil {
			if errObj.Error != "" {
				return errors.New(errObj.Error)
			}
			if errObj.Message != "" {
				return errors.New(errObj.Message)
			}
		}
		return errors.New(string(respBytes))
	}

	if target != nil {
		if err := json.Unmarshal(respBytes, target); err != nil {
			return err
		}
	}

	return nil
}

// AREAS & COURIERS
type AreaItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AdminDiv1  string `json:"administrative_division_level_1_name"`
	AdminDiv2  string `json:"administrative_division_level_2_name"`
	AdminDiv3  string `json:"administrative_division_level_3_name"`
	AdminDiv4  string `json:"administrative_division_level_4_name"`
	PostalCode int    `json:"postal_code"`
}

type areaResponse struct {
	Success bool       `json:"success"`
	Areas   []AreaItem `json:"areas"`
}

func (c *client) SearchArea(ctx context.Context, keyword string) ([]AreaItem, error) {
	endpoint := fmt.Sprintf("%s/v1/maps/areas?countries=ID&input=%s&type=single", c.baseURL, url.QueryEscape(keyword))
	var res areaResponse
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &res); err != nil {
		return nil, err
	}
	return res.Areas, nil
}

type CourierDetail struct {
	AvailableForCashOnDelivery  bool   `json:"available_for_cash_on_delivery"`
	AvailableForProofOfDelivery bool   `json:"available_for_proof_of_delivery"`
	AvailableForInstantWaybill  bool   `json:"available_for_instant_waybill"`
	CouriersName                string `json:"courier_name"`
	CourierCode                 string `json:"courier_code"`
	CourierServiceName          string `json:"courier_service_name"`
	CourierServiceCode          string `json:"courier_service_code"`
	Tier                        string `json:"tier"`
	Description                 string `json:"description"`
	ServiceType                 string `json:"service_type"`
}

type courierListResponse struct {
	Success  bool            `json:"success"`
	Couriers []CourierDetail `json:"couriers"`
}

func (c *client) GetCouriers(ctx context.Context) ([]CourierDetail, error) {
	endpoint := fmt.Sprintf("%s/v1/couriers", c.baseURL)
	var res courierListResponse
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &res); err != nil {
		return nil, err
	}
	return res.Couriers, nil
}

// RATES
type RateItem struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Value       int64  `json:"value"`
	Length      int    `json:"length"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Weight      int    `json:"weight"`
	Quantity    int    `json:"quantity"`
}

type ShippingRatePricing struct {
	AvailableForCashOnDelivery   bool     `json:"available_for_cash_on_delivery"`
	AvailableForProofOfDelivery  bool     `json:"available_for_proof_of_delivery"`
	AvailableForInstantWaybillID bool     `json:"available_for_instant_waybill_id"`
	AvailableCollectionMethod    []string `json:"available_collection_method"`
	Company                      string   `json:"company"`
	CourierName                  string   `json:"courier_name"`
	CourierCode                  string   `json:"courier_code"`
	CourierServiceName           string   `json:"courier_service_name"`
	CourierServiceCode           string   `json:"courier_service_code"`
	Description                  string   `json:"description"`
	Duration                     string   `json:"duration"`
	ShipmentDurationRange        string   `json:"shipment_duration_range"`
	ShipmentDurationUnit         string   `json:"shipment_duration_unit"`
	ServiceType                  string   `json:"service_type"`
	ShippingType                 string   `json:"shipping_type"`
	Price                        int64    `json:"price"`
	Type                         string   `json:"type"`
}

type rateResponse struct {
	Success bool                  `json:"success"`
	Pricing []ShippingRatePricing `json:"pricing"`
}

func (c *client) buildRateCacheKey(destAreaID, couriers string, items []RateItem) string {
	var sb strings.Builder
	sb.WriteString(c.originAreaID)
	sb.WriteString(":")
	sb.WriteString(destAreaID)
	sb.WriteString(":")
	sb.WriteString(couriers)
	sb.WriteString(":")
	for _, it := range items {
		sb.WriteString(it.Name)
		sb.WriteString("|")
		sb.WriteString(strconv.FormatInt(it.Value, 10))
		sb.WriteString("|")
		sb.WriteString(strconv.Itoa(it.Weight))
		sb.WriteString("|")
		sb.WriteString(strconv.Itoa(it.Quantity))
		sb.WriteString("|")
		sb.WriteString(strconv.Itoa(it.Length))
		sb.WriteString("|")
		sb.WriteString(strconv.Itoa(it.Width))
		sb.WriteString("|")
		sb.WriteString(strconv.Itoa(it.Height))
		sb.WriteString(";")
	}
	return sb.String()
}

func (c *client) CalculateRatesByAreaID(ctx context.Context, destAreaID, couriers string, items []RateItem) ([]ShippingRatePricing, error) {
	cacheKey := c.buildRateCacheKey(destAreaID, couriers, items)

	c.rateMu.RLock()
	entry, found := c.rateCache[cacheKey]
	if found && time.Now().Before(entry.expiresAt) {
		c.rateMu.RUnlock()
		cached := make([]ShippingRatePricing, len(entry.pricing))
		copy(cached, entry.pricing)
		return cached, nil
	}
	c.rateMu.RUnlock()

	endpoint := fmt.Sprintf("%s/v1/rates/couriers", c.baseURL)
	body := map[string]interface{}{
		"origin_area_id":      c.originAreaID,
		"destination_area_id": destAreaID,
		"couriers":            couriers,
		"items":               items,
	}
	var res rateResponse
	if err := c.doRequest(ctx, http.MethodPost, endpoint, body, &res); err != nil {
		return nil, err
	}

	if len(res.Pricing) > 0 {
		c.rateMu.Lock()
		if len(c.rateCache) > 1000 {
			now := time.Now()
			for k, v := range c.rateCache {
				if now.After(v.expiresAt) {
					delete(c.rateCache, k)
				}
			}
		}
		c.rateCache[cacheKey] = rateCacheEntry{
			pricing:   res.Pricing,
			expiresAt: time.Now().Add(10 * time.Minute),
		}
		c.rateMu.Unlock()
	}

	return res.Pricing, nil
}

// ORDERS
type CreateOrderByAreaIDRequest struct {
	ShipperContactName      string      `json:"shipper_contact_name,omitempty"`
	ShipperContactPhone     string      `json:"shipper_contact_phone,omitempty"`
	ShipperContactEmail     string      `json:"shipper_contact_email,omitempty"`
	ShipperOrganization     string      `json:"shipper_organization,omitempty"`
	OriginContactName       string      `json:"origin_contact_name"`
	OriginContactPhone      string      `json:"origin_contact_phone"`
	OriginAddress           string      `json:"origin_address"`
	OriginNote              string      `json:"origin_note,omitempty"`
	OriginAreaID            string      `json:"origin_area_id,omitempty"`
	OriginPostalCode        int         `json:"origin_postal_code,omitempty"`
	DestinationContactName  string      `json:"destination_contact_name"`
	DestinationContactPhone string      `json:"destination_contact_phone"`
	DestinationContactEmail string      `json:"destination_contact_email,omitempty"`
	DestinationAddress      string      `json:"destination_address"`
	DestinationNote         string      `json:"destination_note,omitempty"`
	DestinationAreaID       string      `json:"destination_area_id"`
	DestinationPostalCode   int         `json:"destination_postal_code,omitempty"`
	CourierCompany          string      `json:"courier_company"`
	CourierType             string      `json:"courier_type"`
	CourierInsurance        int64       `json:"courier_insurance,omitempty"`
	DeliveryType            string      `json:"delivery_type"`
	OrderNote               string      `json:"order_note,omitempty"`
	Metadata                interface{} `json:"metadata,omitempty"`
	Items                   []RateItem  `json:"items"`
}

type OrderResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      string `json:"id"`
	Courier struct {
		TrackingID  string `json:"tracking_id"`
		WaybillID   string `json:"waybill_id"`
		Company     string `json:"company"`
		Type        string `json:"type"`
		Link        string `json:"link"`
		TrackingURL string `json:"tracking_url"`
	} `json:"courier"`
	Origin struct {
		ContactName string `json:"contact_name"`
		Address     string `json:"address"`
		PostalCode  int    `json:"postal_code"`
	} `json:"origin"`
	Destination struct {
		ContactName string `json:"contact_name"`
		Address     string `json:"address"`
		PostalCode  int    `json:"postal_code"`
	} `json:"destination"`
	Price  int64  `json:"price"`
	Status string `json:"status"`
}

type CancelOrderResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ID           string `json:"id"`
	Status       string `json:"status"`
	Cancellation struct {
		Reason string `json:"reason"`
	} `json:"cancellation"`
}

func (c *client) CreateOrderByAreaID(ctx context.Context, req CreateOrderByAreaIDRequest) (*OrderResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/orders", c.baseURL)
	if req.OriginAreaID == "" {
		req.OriginAreaID = c.originAreaID
	}
	var res OrderResponse
	if err := c.doRequest(ctx, http.MethodPost, endpoint, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) GetOrderByID(ctx context.Context, orderID string) (*OrderResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/orders/%s", c.baseURL, orderID)
	var res OrderResponse
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) CancelOrder(ctx context.Context, orderID string, reason string) (*CancelOrderResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/orders/%s/cancel", c.baseURL, orderID)
	body := map[string]string{
		"reason": reason,
	}
	var res CancelOrderResponse
	if err := c.doRequest(ctx, http.MethodPost, endpoint, body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// TRACKING
type TrackingHistory struct {
	Note      string `json:"note"`
	UpdatedAt string `json:"updated_at"`
	Status    string `json:"status"`
}

type CourierInfo struct {
	Company     string  `json:"company"`
	Name        string  `json:"name"`
	DriverName  *string `json:"driver_name"`
	DriverPhone *string `json:"driver_phone"`
	Type        string  `json:"type"`
	Link        *string `json:"link"`
}

type TrackingDetailResult struct {
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	WaybillID   string            `json:"waybill_id"`
	CourierCode string            `json:"courier_code"`
	CourierName string            `json:"courier_name"`
	Courier     CourierInfo       `json:"courier"`
	Status      string            `json:"status"`
	History     []TrackingHistory `json:"history"`
}

func (c *client) TrackByID(ctx context.Context, trackingID string) (*TrackingDetailResult, error) {
	endpoint := fmt.Sprintf("%s/v1/trackings/%s", c.baseURL, trackingID)
	var res TrackingDetailResult
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &res); err != nil {
		return nil, err
	}
	if res.CourierCode == "" {
		res.CourierCode = res.Courier.Company
	}
	if res.CourierName == "" {
		res.CourierName = res.Courier.Name
	}
	return &res, nil
}

func (c *client) TrackWaybill(ctx context.Context, waybillID, courierCode string) (*TrackingDetailResult, error) {
	endpoint := fmt.Sprintf("%s/v1/trackings/%s/couriers/%s", c.baseURL, waybillID, courierCode)
	var res TrackingDetailResult
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &res); err != nil {
		return nil, err
	}
	if res.CourierCode == "" {
		res.CourierCode = res.Courier.Company
		if res.CourierCode == "" {
			res.CourierCode = courierCode
		}
	}
	if res.CourierName == "" {
		res.CourierName = res.Courier.Name
		if res.CourierName == "" {
			res.CourierName = strings.ToUpper(courierCode)
		}
	}
	return &res, nil
}

// WEBHOOK PAYLOAD
type WebhookTrackingPayload struct {
	Event             string                 `json:"event"`
	ID                string                 `json:"id"`
	CourierTrackingID string                 `json:"courier_tracking_id"`
	TrackingID        string                 `json:"tracking_id"`
	OrderID           string                 `json:"order_id"`
	WaybillID         string                 `json:"waybill_id"`
	CourierCode       string                 `json:"courier_code"`
	Status            string                 `json:"status"`
	OrderStatus       string                 `json:"order_status"`
	Note              string                 `json:"note"`
	UpdatedAt         interface{}            `json:"updated_at"`
	Metadata          map[string]interface{} `json:"metadata"`
	Data              *WebhookDataPayload    `json:"data,omitempty"`
}

type WebhookDataPayload struct {
	ID                string `json:"id"`
	OrderID           string `json:"order_id"`
	Status            string `json:"status"`
	OrderStatus       string `json:"order_status"`
	WaybillID         string `json:"waybill_id"`
	CourierTrackingID string `json:"courier_tracking_id"`
	TrackingID        string `json:"tracking_id"`
	CourierCode       string `json:"courier_code"`
}

func (p *WebhookTrackingPayload) GetOrderID() string {
	if p.OrderID != "" {
		return p.OrderID
	}
	if p.ID != "" {
		return p.ID
	}
	if p.Data != nil {
		if p.Data.OrderID != "" {
			return p.Data.OrderID
		}
		if p.Data.ID != "" {
			return p.Data.ID
		}
	}
	if p.Metadata != nil {
		if inv, ok := p.Metadata["invoice_number"].(string); ok && inv != "" {
			return inv
		}
		if oid, ok := p.Metadata["order_id"].(string); ok && oid != "" {
			return oid
		}
	}
	return ""
}

func (p *WebhookTrackingPayload) GetTrackingID() string {
	if p.CourierTrackingID != "" {
		return p.CourierTrackingID
	}
	if p.TrackingID != "" {
		return p.TrackingID
	}
	if p.Data != nil {
		if p.Data.CourierTrackingID != "" {
			return p.Data.CourierTrackingID
		}
		if p.Data.TrackingID != "" {
			return p.Data.TrackingID
		}
	}
	return ""
}

func (p *WebhookTrackingPayload) GetWaybillID() string {
	if p.WaybillID != "" {
		return p.WaybillID
	}
	if p.Data != nil && p.Data.WaybillID != "" {
		return p.Data.WaybillID
	}
	return ""
}

func (p *WebhookTrackingPayload) GetStatus() string {
	if p.Status != "" {
		return p.Status
	}
	if p.OrderStatus != "" {
		return p.OrderStatus
	}
	if p.Data != nil {
		if p.Data.Status != "" {
			return p.Data.Status
		}
		if p.Data.OrderStatus != "" {
			return p.Data.OrderStatus
		}
	}
	return ""
}
