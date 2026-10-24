package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPaymentMethodUsecase struct {
	mock.Mock
}

func (m *MockPaymentMethodUsecase) GetAll(ctx context.Context, onlyActive bool, amount int) (*dto.PaymentMethodsResponse, error) {
	args := m.Called(ctx, onlyActive, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PaymentMethodsResponse), args.Error(1)
}

func (m *MockPaymentMethodUsecase) GetByID(ctx context.Context, id uint) (*dto.PaymentMethodDetailResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PaymentMethodDetailResponse), args.Error(1)
}

func (m *MockPaymentMethodUsecase) Update(ctx context.Context, id uint, req dto.UpdatePaymentMethodRequest) (*dto.PaymentMethodDetailResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PaymentMethodDetailResponse), args.Error(1)
}

func setupPaymentMethodApp(uc *MockPaymentMethodUsecase) *fiber.App {
	app := fiber.New()
	v := validator.New()

	dummyAuth := func(c fiber.Ctx) error {
		return c.Next()
	}
	dummyAdmin := func(c fiber.Ctx) error {
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewPaymentMethodHandler(api, uc, v, dummyAuth, dummyAdmin)

	return app
}

func TestPaymentMethodHandler(t *testing.T) {
	t.Run("GET /api/v1/payment-methods returns 200", func(t *testing.T) {
		mockUC := new(MockPaymentMethodUsecase)
		app := setupPaymentMethodApp(mockUC)

		expected := &dto.PaymentMethodsResponse{
			Groups: []dto.PaymentMethodGroupResponse{
				{
					Category: "virtual_account",
					Name:     "Virtual Account",
					Methods: []dto.PaymentMethodItemResponse{
						{
							ID:       1,
							Code:     "bca_va",
							Name:     "BCA Virtual Account",
							IsActive: true,
						},
					},
				},
			},
		}

		mockUC.On("GetAll", mock.Anything, false, 0).Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockUC.AssertExpectations(t)
	})

	t.Run("GET /api/v1/payment-methods/:id returns 200", func(t *testing.T) {
		mockUC := new(MockPaymentMethodUsecase)
		app := setupPaymentMethodApp(mockUC)

		expected := &dto.PaymentMethodDetailResponse{
			ID:        1,
			Code:      "bca_va",
			Name:      "BCA Virtual Account",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockUC.On("GetByID", mock.Anything, uint(1)).Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/1", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockUC.AssertExpectations(t)
	})

	t.Run("GET /api/v1/payment-methods/:id returns 404 when not found", func(t *testing.T) {
		mockUC := new(MockPaymentMethodUsecase)
		app := setupPaymentMethodApp(mockUC)

		mockUC.On("GetByID", mock.Anything, uint(99)).Return(nil, entity.ErrPaymentMethodNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-methods/99", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockUC.AssertExpectations(t)
	})

	t.Run("PUT /api/v1/payment-methods/:id toggles maintenance active state", func(t *testing.T) {
		mockUC := new(MockPaymentMethodUsecase)
		app := setupPaymentMethodApp(mockUC)

		body, _ := json.Marshal(map[string]interface{}{
			"is_active": false,
		})

		expected := &dto.PaymentMethodDetailResponse{
			ID:        1,
			Code:      "qris",
			Name:      "QRIS",
			IsActive:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockUC.On("Update", mock.Anything, uint(1), mock.MatchedBy(func(r dto.UpdatePaymentMethodRequest) bool {
			return r.IsActive != nil && *r.IsActive == false
		})).Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodPut, "/api/v1/payment-methods/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(respBody), "payment method updated successfully")

		mockUC.AssertExpectations(t)
	})
}
