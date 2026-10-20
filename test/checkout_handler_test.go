package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCheckoutUsecase struct {
	mock.Mock
}

func (m *MockCheckoutUsecase) PreviewCheckout(ctx context.Context, userID uint, input usecase.PreviewCheckoutInput) (*usecase.PreviewCheckoutResult, error) {
	args := m.Called(ctx, userID, input)
	if res := args.Get(0); res != nil {
		return res.(*usecase.PreviewCheckoutResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCheckoutUsecase) CreateOrder(ctx context.Context, userID uint, input usecase.CreateOrderInput) (*usecase.CreateOrderResult, error) {
	args := m.Called(ctx, userID, input)
	if res := args.Get(0); res != nil {
		return res.(*usecase.CreateOrderResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCheckoutUsecase) HandleMidtransWebhook(ctx context.Context, input usecase.MidtransWebhookInput) error {
	return m.Called(ctx, input).Error(0)
}

func setupCheckoutApp(mockUC *MockCheckoutUsecase, authenticated bool) *fiber.App {
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

	api := app.Group("/api/v1")
	handler.NewCheckoutHandler(api, mockUC, validate, authMiddleware)
	return app
}

func TestCheckoutHandler_PreviewCheckout(t *testing.T) {
	addrID := uint(5)
	reqPayload := dto.CheckoutPreviewRequest{
		CartItemIDs:    []uint{1, 2},
		AddressID:      &addrID,
		Courier:        "jne",
		CourierService: "reg",
		PaymentMethod:  "bca_va",
	}
	body, _ := json.Marshal(reqPayload)

	t.Run("success preview returns 200", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, true)

		expectedRes := &usecase.PreviewCheckoutResult{
			Address: &entity.Address{
				ID:           5,
				ReceiverName: "Budi Santoso",
				Phone:        "08123456789",
				AddressLine:  "Jl. Sudirman No. 1",
				ProvinceName: "DKI Jakarta",
				CityName:     "Jakarta Selatan",
				PostalCode:   "12190",
			},
			ItemsSubtotal:    200000,
			TotalWeightGrams: 1000,
			Courier:          "jne",
			CourierService:   "reg",
			ShippingCost:     18000,
			PaymentMethod:    "bca_va",
			AdminFee:         5000,
			TotalPayment:     223000,
		}

		mockUC.On("PreviewCheckout", mock.Anything, uint(1), mock.MatchedBy(func(in usecase.PreviewCheckoutInput) bool {
			return in.AddressID != nil && *in.AddressID == 5 && in.Courier == "jne"
		})).Return(expectedRes, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("unauthorized user returns 401", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, false)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("validation error on missing required fields returns 400", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, true)

		invalidBody := []byte(`{"cart_item_ids": [], "courier": ""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", bytes.NewReader(invalidBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("address not found returns 404", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, true)

		mockUC.On("PreviewCheckout", mock.Anything, uint(1), mock.Anything).
			Return(nil, entity.ErrAddressNotFound).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestCheckoutHandler_CreateOrder(t *testing.T) {
	addrID := uint(5)
	reqPayload := dto.CreateOrderRequest{
		CartItemIDs:    []uint{1},
		AddressID:      &addrID,
		Courier:        "jne",
		CourierService: "reg",
		PaymentMethod:  "bca_va",
	}
	body, _ := json.Marshal(reqPayload)

	t.Run("success create order returns 201", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, true)

		bank := "bca"
		va := "123456789012"
		expectedRes := &usecase.CreateOrderResult{
			InvoiceNumber:   "INV-20261017-000101",
			Status:          "waiting_payment",
			TotalAmount:     223000,
			PaymentDeadline: time.Now().Add(24 * time.Hour),
		}
		expectedRes.PaymentInfo.PaymentType = "bank_transfer"
		expectedRes.PaymentInfo.Bank = &bank
		expectedRes.PaymentInfo.VANumber = &va

		mockUC.On("CreateOrder", mock.Anything, uint(1), mock.Anything).Return(expectedRes, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/order", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("insufficient book stock returns 400", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, true)

		mockUC.On("CreateOrder", mock.Anything, uint(1), mock.Anything).
			Return(nil, entity.ErrInsufficientStock).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/order", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("payment gateway error returns 502 Bad Gateway", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, true)

		mockUC.On("CreateOrder", mock.Anything, uint(1), mock.Anything).
			Return(nil, entity.ErrMidtransChargeFailed).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/order", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("invalid payment method returns 400 Bad Request", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, true)

		mockUC.On("CreateOrder", mock.Anything, uint(1), mock.Anything).
			Return(nil, entity.ErrInvalidPaymentMethod).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/order", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestCheckoutHandler_HandleMidtransWebhook(t *testing.T) {
	webhookBody := []byte(`{
		"order_id": "INV-20261017-000101",
		"status_code": "200",
		"gross_amount": "223000.00",
		"signature_key": "valid_signature",
		"transaction_status": "settlement"
	}`)

	t.Run("success handle webhook returns 200 ok", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, false)

		mockUC.On("HandleMidtransWebhook", mock.Anything, mock.MatchedBy(func(in usecase.MidtransWebhookInput) bool {
			return in.OrderID == "INV-20261017-000101" && in.TransactionStatus == "settlement"
		})).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/midtrans/webhook", bytes.NewReader(webhookBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("invalid webhook signature returns 403 Forbidden", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, false)

		mockUC.On("HandleMidtransWebhook", mock.Anything, mock.Anything).
			Return(errors.New("invalid midtrans webhook signature")).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/midtrans/webhook", bytes.NewReader(webhookBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}
