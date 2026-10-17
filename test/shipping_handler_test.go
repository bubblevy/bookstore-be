package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockShippingUsecase struct {
	mock.Mock
}

func (m *MockShippingUsecase) SearchAreas(ctx context.Context, keyword string) ([]biteship.AreaItem, error) {
	args := m.Called(ctx, keyword)
	if res := args.Get(0); res != nil {
		return res.([]biteship.AreaItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShippingUsecase) GetCouriers(ctx context.Context) ([]biteship.CourierDetail, error) {
	args := m.Called(ctx)
	if res := args.Get(0); res != nil {
		return res.([]biteship.CourierDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShippingUsecase) GetShippingRates(ctx context.Context, userID uint, input usecase.GetShippingRatesInput) (*usecase.ShippingRatesResult, error) {
	args := m.Called(ctx, userID, input)
	if res := args.Get(0); res != nil {
		return res.(*usecase.ShippingRatesResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShippingUsecase) FulfillOrder(ctx context.Context, invoiceNumber string, orderNote string) (*usecase.FulfillOrderResult, error) {
	args := m.Called(ctx, invoiceNumber, orderNote)
	if res := args.Get(0); res != nil {
		return res.(*usecase.FulfillOrderResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShippingUsecase) GetOrderTracking(ctx context.Context, userID uint, invoiceNumber string) (*biteship.TrackingDetailResult, error) {
	args := m.Called(ctx, userID, invoiceNumber)
	if res := args.Get(0); res != nil {
		return res.(*biteship.TrackingDetailResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShippingUsecase) ProcessReturn(ctx context.Context, invoiceNumber string, reason string) error {
	return m.Called(ctx, invoiceNumber, reason).Error(0)
}

func (m *MockShippingUsecase) HandleWebhook(ctx context.Context, payload biteship.WebhookTrackingPayload) error {
	return m.Called(ctx, payload).Error(0)
}

func setupShippingApp(mockUC *MockShippingUsecase, authenticated bool, isAdmin bool) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	authMiddleware := func(c fiber.Ctx) error {
		if !authenticated {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    fiber.StatusUnauthorized,
				"status":  false,
				"message": "unauthorized",
			})
		}
		c.Locals("user_id", uint(1))
		return c.Next()
	}

	adminMiddleware := func(c fiber.Ctx) error {
		if !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    fiber.StatusForbidden,
				"status":  false,
				"message": "forbidden: admin access required",
			})
		}
		return c.Next()
	}

	webhookAuthMiddleware := func(c fiber.Ctx) error {
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewShippingHandler(api, mockUC, validate, authMiddleware, adminMiddleware, webhookAuthMiddleware)
	return app
}

func TestShippingHandler_SearchAreas(t *testing.T) {
	t.Run("success search returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, false, false)

		areas := []biteship.AreaItem{
			{
				ID:         "IDNP6IDNC148",
				Name:       "Kebayoran Baru, Jakarta Selatan",
				AdminDiv1:  "DKI Jakarta",
				AdminDiv2:  "Jakarta Selatan",
				AdminDiv3:  "Kebayoran Baru",
				PostalCode: 12190,
			},
		}
		mockUC.On("SearchAreas", mock.Anything, "jakarta").Return(areas, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/areas?search=jakarta", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("query length less than 3 returns 400", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, false, false)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/areas?search=jk", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestShippingHandler_GetCouriers(t *testing.T) {
	t.Run("success get couriers returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, false, false)

		couriers := []biteship.CourierDetail{
			{
				CourierCode:        "jne",
				CouriersName:       "JNE",
				CourierServiceCode: "reg",
				CourierServiceName: "Reguler",
				Tier:               "standard",
				ServiceType:        "standard",
			},
		}
		mockUC.On("GetCouriers", mock.Anything).Return(couriers, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/couriers", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestShippingHandler_GetShippingRates(t *testing.T) {
	addrID := uint(5)
	payload := dto.GetShippingRatesRequest{
		CartItemIDs: []uint{1, 2},
		AddressID:   &addrID,
	}
	body, _ := json.Marshal(payload)

	t.Run("success returns rates 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, true, false)

		expectedRes := &usecase.ShippingRatesResult{
			TotalWeightGrams: 1000,
			DestinationArea:  "Jl. Pemuda No. 45, Kel. Kramat Pela, Kec. Kebayoran Baru, Jakarta Selatan, DKI Jakarta 12250",
			Options: []usecase.CourierOptionResult{
				{
					CourierCode:        "jne",
					CourierName:        "JNE",
					CourierServiceName: "Reguler",
					CourierServiceCode: "reg",
					Price:              20000,
					ETD:                "1-2 days",
				},
			},
		}

		mockUC.On("GetShippingRates", mock.Anything, uint(1), mock.MatchedBy(func(in usecase.GetShippingRatesInput) bool {
			return len(in.CartItemIDs) == 2 && in.AddressID != nil && *in.AddressID == 5
		})).Return(expectedRes, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/rates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("cart empty returns 400", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, true, false)

		mockUC.On("GetShippingRates", mock.Anything, uint(1), mock.Anything).
			Return(nil, entity.ErrCartEmpty).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/rates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestShippingHandler_FulfillOrder(t *testing.T) {
	invoice := "INV-20261017-000101"

	t.Run("admin success fulfill order returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, true, true)

		waybill := "SOCAG001"
		trackURL := "/orders/" + invoice + "/tracking"
		fulfillRes := &usecase.FulfillOrderResult{
			ShipmentID:      1,
			InvoiceNumber:   invoice,
			BiteshipOrderID: "bite-12345",
			WaybillNumber:   &waybill,
			Status:          entity.ShippingStatusAllocated,
			Courier:         "jne",
			CourierService:  "reg",
			TrackingURL:     &trackURL,
		}

		mockUC.On("FulfillOrder", mock.Anything, invoice, "").Return(fulfillRes, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+invoice+"/fulfill", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("non-admin returns 403 Forbidden", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, true, false)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+invoice+"/fulfill", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestShippingHandler_TrackOrder(t *testing.T) {
	invoice := "INV-20261017-000101"

	t.Run("success track order returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, true, false)

		trackDetail := &biteship.TrackingDetailResult{
			WaybillID:   "SOCAG001",
			CourierCode: "jne",
			Status:      "allocated",
		}
		mockUC.On("GetOrderTracking", mock.Anything, uint(1), invoice).Return(trackDetail, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+invoice+"/tracking", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("tracking number not ready returns 400", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, true, false)

		mockUC.On("GetOrderTracking", mock.Anything, uint(1), invoice).
			Return(nil, entity.ErrTrackingNumberNotReady).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+invoice+"/tracking", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestShippingHandler_ProcessReturn(t *testing.T) {
	invoice := "INV-20261017-000101"

	t.Run("admin success process return returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, true, true)

		mockUC.On("ProcessReturn", mock.Anything, invoice, "Customer rejected package").Return(nil).Once()

		reqBody := []byte(`{"reason": "Customer rejected package"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+invoice+"/return", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestShippingHandler_HandleWebhook(t *testing.T) {
	t.Run("success process webhook returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingApp(mockUC, false, false)

		webhookBody := []byte(`{
			"order_id": "bite-12345",
			"status": "delivered",
			"waybill_id": "SOCAG001"
		}`)

		mockUC.On("HandleWebhook", mock.Anything, mock.MatchedBy(func(p biteship.WebhookTrackingPayload) bool {
			return p.OrderID == "bite-12345" && p.Status == "delivered"
		})).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/webhook", bytes.NewReader(webhookBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}
