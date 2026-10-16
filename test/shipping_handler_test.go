package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
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

func (m *MockShippingUsecase) FulfillOrder(ctx context.Context, orderID uint, orderNote string) (*usecase.FulfillOrderResult, error) {
	args := m.Called(ctx, orderID, orderNote)
	if res := args.Get(0); res != nil {
		return res.(*usecase.FulfillOrderResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShippingUsecase) GetOrderTracking(ctx context.Context, userID, orderID uint) (*biteship.TrackingDetailResult, error) {
	args := m.Called(ctx, userID, orderID)
	if res := args.Get(0); res != nil {
		return res.(*biteship.TrackingDetailResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShippingUsecase) HandleWebhook(ctx context.Context, payload biteship.WebhookTrackingPayload) error {
	return m.Called(ctx, payload).Error(0)
}

func setupShippingHandlerApp(mockUC *MockShippingUsecase, authenticated, isAdmin bool) *fiber.App {
	app := fiber.New()

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
				"message": "forbidden",
			})
		}
		return c.Next()
	}

	handler.NewShippingHandler(app, mockUC, authMiddleware, adminMiddleware)
	return app
}

func TestShippingHandler_SearchAreas(t *testing.T) {
	t.Run("success search areas returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, false, false)

		areas := []biteship.AreaItem{
			{ID: "IDNP6IDNC148", Name: "Kebayoran Baru", PostalCode: 12190},
		}
		mockUC.On("SearchAreas", mock.Anything, "kebayoran").Return(areas, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/shipping/areas?search=kebayoran", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("error when search keyword too short returns 400", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, false, false)

		req := httptest.NewRequest(http.MethodGet, "/shipping/areas?search=ke", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestShippingHandler_FulfillOrder(t *testing.T) {
	t.Run("success admin fulfill order returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, true, true)

		orderID := uint(5)
		waybill := "WAYBILL123"
		expectedRes := &usecase.FulfillOrderResult{
			ShipmentID:      1,
			OrderID:         orderID,
			BiteshipOrderID: "biteship-123",
			WaybillNumber:   &waybill,
			Status:          entity.ShippingStatusAllocated,
			Courier:         "jne",
			CourierService:  "reg",
		}

		mockUC.On("FulfillOrder", mock.Anything, orderID, "Handle with care").
			Return(expectedRes, nil).Once()

		body := []byte(`{"order_note":"Handle with care"}`)
		req := httptest.NewRequest(http.MethodPost, "/admin/orders/5/fulfill", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("failed when non-admin user tries fulfill returns 403", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, true, false) // Not admin

		req := httptest.NewRequest(http.MethodPost, "/admin/orders/5/fulfill", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("invalid order id param returns 400", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, true, true)

		req := httptest.NewRequest(http.MethodPost, "/admin/orders/abc/fulfill", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestShippingHandler_TrackOrder(t *testing.T) {
	t.Run("success track order returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, true, false)

		orderID := uint(10)
		expected := &biteship.TrackingDetailResult{
			Success:   true,
			WaybillID: "WAYBILL123",
			Status:    "in_transit",
		}
		mockUC.On("GetOrderTracking", mock.Anything, uint(1), orderID).
			Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/orders/10/tracking", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("order not found returns 404", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, true, false)

		orderID := uint(99)
		mockUC.On("GetOrderTracking", mock.Anything, uint(1), orderID).
			Return(nil, entity.ErrOrderNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/orders/99/tracking", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestShippingHandler_HandleWebhook(t *testing.T) {
	t.Run("success handle webhook returns 200", func(t *testing.T) {
		mockUC := new(MockShippingUsecase)
		app := setupShippingHandlerApp(mockUC, false, false)

		mockUC.On("HandleWebhook", mock.Anything, mock.MatchedBy(func(p biteship.WebhookTrackingPayload) bool {
			return p.OrderID == "biteship-123" && p.Status == "delivered"
		})).Return(nil).Once()

		payload := biteship.WebhookTrackingPayload{
			OrderID:   "biteship-123",
			Status:    "delivered",
			WaybillID: "WAYBILL123",
			UpdatedAt: time.Now(),
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/shipping/webhook", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}
