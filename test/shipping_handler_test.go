package test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bubblevy/bookstore/config"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/pkg/rajaongkir"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// RAJAONGKIR CLIENT UNIT TESTS (MOCK HTTP SERVER)
// ============================================================================

func TestRajaOngkirClient_SearchDestination_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/destination/domestic-destination", r.URL.Path)
		assert.Equal(t, "welahan", r.URL.Query().Get("search"))
		assert.Equal(t, "mock-api-key", r.Header.Get("key"))

		resp := rajaongkir.BaseResponse[[]rajaongkir.DomesticDestination]{
			Data: []rajaongkir.DomesticDestination{
				{
					ID:          65216,
					Label:       "KALIPUCANG WETAN, WELAHAN, JEPARA, JAWA TENGAH, 59464",
					Subdistrict: "KALIPUCANG WETAN",
					District:    "WELAHAN",
					City:        "JEPARA",
					Province:    "JAWA TENGAH",
					ZipCode:     "59464",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	client := rajaongkir.NewClient(config.RajaOngkirConfig{
		APIKey:   "mock-api-key",
		BaseURL:  mockServer.URL,
		OriginID: 65216,
	})

	results, err := client.SearchDestination(context.Background(), "welahan", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 65216, results[0].ID)
	assert.Equal(t, "WELAHAN", results[0].District)
}

func TestRajaOngkirClient_CalculateCost_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/calculate/domestic-cost", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		_ = r.ParseForm()
		assert.Equal(t, "65216", r.Form.Get("origin"))
		assert.Equal(t, "19601", r.Form.Get("destination"))
		assert.Equal(t, "1200", r.Form.Get("weight"))

		resp := rajaongkir.BaseResponse[[]rajaongkir.ShippingCostResult]{
			Data: []rajaongkir.ShippingCostResult{
				{
					Name:        "Jalur Nugraha Ekakurir (JNE)",
					Code:        "jne",
					Service:     "REG",
					Description: "Layanan Reguler",
					Cost:        18000,
					ETD:         "2-3",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	client := rajaongkir.NewClient(config.RajaOngkirConfig{
		APIKey:   "mock-api-key",
		BaseURL:  mockServer.URL,
		OriginID: 65216,
	})

	req := rajaongkir.CalculateDomesticCostRequest{
		DestinationID: 19601,
		WeightGrams:   1200,
	}

	results, err := client.CalculateCost(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "jne", results[0].Code)
	assert.Equal(t, 18000, results[0].Cost)
}

// ============================================================================
// SHIPPING HTTP HANDLER TESTS
// ============================================================================

func setupShippingApp(mockServerURL string) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	client := rajaongkir.NewClient(config.RajaOngkirConfig{
		APIKey:   "mock-key",
		BaseURL:  mockServerURL,
		OriginID: 65216,
	})

	api := app.Group("/api/v1")
	handler.NewShippingHandler(api, client, validate)
	return app
}

func TestShippingHandler_SearchDestinations_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := rajaongkir.BaseResponse[[]rajaongkir.DomesticDestination]{
			Data: []rajaongkir.DomesticDestination{
				{
					ID:       65216,
					Label:    "KALIPUCANG WETAN, WELAHAN, JEPARA, JAWA TENGAH, 59464",
					District: "WELAHAN",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	app := setupShippingApp(mockServer.URL)

	req := httptest.NewRequest("GET", "/api/v1/shipping/destinations?search=welahan", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)

	var res struct {
		Code    int                              `json:"code"`
		Status  bool                             `json:"status"`
		Message string                           `json:"message"`
		Data    []rajaongkir.DomesticDestination `json:"data"`
	}
	err = json.Unmarshal(body, &res)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.Code)
	assert.True(t, res.Status)
	assert.Len(t, res.Data, 1)
	assert.Equal(t, 65216, res.Data[0].ID)
}

func TestShippingHandler_SearchDestinations_QueryTooShort(t *testing.T) {
	app := setupShippingApp("http://dummy-url")

	req := httptest.NewRequest("GET", "/api/v1/shipping/destinations?search=we", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestShippingHandler_CheckCost_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := rajaongkir.BaseResponse[[]rajaongkir.ShippingCostResult]{
			Data: []rajaongkir.ShippingCostResult{
				{
					Code:    "jne",
					Service: "REG",
					Cost:    15000,
					ETD:     "2-3",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	app := setupShippingApp(mockServer.URL)

	reqBody := `{"destination_id": 19601, "weight_grams": 1000}`
	req := httptest.NewRequest("POST", "/api/v1/shipping/cost", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)

	var res struct {
		Code    int                             `json:"code"`
		Status  bool                            `json:"status"`
		Message string                          `json:"message"`
		Data    []rajaongkir.ShippingCostResult `json:"data"`
	}
	err = json.Unmarshal(body, &res)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.Code)
	assert.True(t, res.Status)
	assert.Len(t, res.Data, 1)
	assert.Equal(t, 15000, res.Data[0].Cost)
}

func TestShippingHandler_CheckCost_ValidationError(t *testing.T) {
	app := setupShippingApp("http://dummy-url")

	// Payload tanpa destination_id (wajib > 0)
	reqBody := `{"destination_id": 0, "weight_grams": 1000}`
	req := httptest.NewRequest("POST", "/api/v1/shipping/cost", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
