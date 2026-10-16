package test

import (
	"bytes"
	"context"
	"encoding/json"
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
			ItemsSubtotal: 200000,
			ShippingCost:  18000,
			AdminFee:      4000,
			TotalPayment:  222000,
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

		// cart_item_ids kosong atau courier tidak diisi
		invalidBody := []byte(`{"cart_item_ids": [], "courier": ""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/preview", bytes.NewReader(invalidBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
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

		va := "1234567890"
		expectedRes := &usecase.CreateOrderResult{
			InvoiceNumber:   "inv-20261016-000101",
			Status:          "waiting_payment",
			TotalAmount:     222000,
			PaymentDeadline: time.Now().Add(24 * time.Hour),
			PaymentInfo: usecase.PaymentInfoResult{
				PaymentType: "bank_transfer",
				VANumber:    &va,
			},
		}

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
	})
}

func TestCheckoutHandler_HandleMidtransWebhook(t *testing.T) {
	webhookBody := []byte(`{
		"order_id": "inv-20261016-000101",
		"status_code": "200",
		"gross_amount": "222000.00",
		"signature_key": "valid_signature",
		"transaction_status": "settlement"
	}`)

	t.Run("success handle webhook returns 200 ok", func(t *testing.T) {
		mockUC := new(MockCheckoutUsecase)
		app := setupCheckoutApp(mockUC, false)

		mockUC.On("HandleMidtransWebhook", mock.Anything, mock.MatchedBy(func(in usecase.MidtransWebhookInput) bool {
			return in.OrderID == "inv-20261016-000101" && in.TransactionStatus == "settlement"
		})).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/midtrans/webhook", bytes.NewReader(webhookBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}
