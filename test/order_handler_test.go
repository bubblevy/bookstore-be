package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOrderUsecase struct {
	mock.Mock
}

func (m *MockOrderUsecase) GetUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error) {
	args := m.Called(ctx, userID, statusFilter)
	if res := args.Get(0); res != nil {
		return res.([]entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderUsecase) GetOrderDetail(ctx context.Context, userID uint, invoiceNumber string) (*entity.Order, error) {
	args := m.Called(ctx, userID, invoiceNumber)
	if res := args.Get(0); res != nil {
		return res.(*entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupOrderApp(mockUC *MockOrderUsecase, authenticated bool, currentUserID uint) *fiber.App {
	app := fiber.New()

	authMiddleware := func(c fiber.Ctx) error {
		if !authenticated {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    fiber.StatusUnauthorized,
				"status":  false,
				"message": "unauthorized",
			})
		}
		c.Locals("user_id", currentUserID)
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewOrderHandler(api, mockUC, authMiddleware)
	return app
}

func TestOrderHandler_GetUserOrders(t *testing.T) {
	userID := uint(1)

	t.Run("success returns order list with calculated extra items and quantities", func(t *testing.T) {
		mockUC := new(MockOrderUsecase)
		app := setupOrderApp(mockUC, true, userID)

		cover1 := "book1.jpg"
		cover2 := "book2.jpg"
		orders := []entity.Order{
			{
				ID:            101,
				InvoiceNumber: "INV-20261017-000101",
				Status:        entity.OrderStatusWaitingPayment,
				TotalAmount:   250000,
				CreatedAt:     time.Now(),
				Items: []entity.OrderItem{
					{Title: "Clean Architecture", CoverImage: &cover1, Quantity: 2},
					{Title: "Refactoring", CoverImage: &cover2, Quantity: 1},
				},
			},
		}

		mockUC.On("GetUserOrders", mock.Anything, userID, "").Return(orders, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("success with query status filter", func(t *testing.T) {
		mockUC := new(MockOrderUsecase)
		app := setupOrderApp(mockUC, true, userID)

		mockUC.On("GetUserOrders", mock.Anything, userID, "paid").Return([]entity.Order{}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?status=paid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("unauthorized user returns 401", func(t *testing.T) {
		mockUC := new(MockOrderUsecase)
		app := setupOrderApp(mockUC, false, 0)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestOrderHandler_GetOrderDetail(t *testing.T) {
	userID := uint(1)
	invoice := "INV-20261017-000101"

	t.Run("success returns order detail 200", func(t *testing.T) {
		mockUC := new(MockOrderUsecase)
		app := setupOrderApp(mockUC, true, userID)

		bank := "bca"
		va := "123456789"
		order := &entity.Order{
			ID:             101,
			UserID:         userID,
			InvoiceNumber:  invoice,
			Status:         entity.OrderStatusWaitingPayment,
			TotalAmount:    250000,
			Courier:        "jne",
			CourierService: "reg",
			Items: []entity.OrderItem{
				{BookID: 10, Title: "Clean Architecture", Quantity: 1, Price: 250000, Subtotal: 250000},
			},
			Payment: &entity.Payment{
				PaymentType: "bank_transfer",
				Bank:        &bank,
				VANumber:    &va,
				Status:      entity.PaymentStatusPending,
			},
			Shipment: &entity.Shipment{
				Status: entity.ShippingStatusAllocated,
			},
		}

		mockUC.On("GetOrderDetail", mock.Anything, userID, invoice).Return(order, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+invoice, nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("order not found returns 404", func(t *testing.T) {
		mockUC := new(MockOrderUsecase)
		app := setupOrderApp(mockUC, true, userID)

		mockUC.On("GetOrderDetail", mock.Anything, userID, "INV-NOT-FOUND").
			Return(nil, entity.ErrOrderNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/INV-NOT-FOUND", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}
