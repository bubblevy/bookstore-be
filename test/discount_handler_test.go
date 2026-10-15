package test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDiscountUsecase struct {
	mock.Mock
}

func (m *MockDiscountUsecase) Create(ctx context.Context, input usecase.CreateDiscountInput) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.DiscountResult), args.Error(1)
}

func (m *MockDiscountUsecase) FindAll(ctx context.Context, filter repository.DiscountFilter, pq pagination.Query) ([]usecase.DiscountListItem, pagination.Meta, error) {
	args := m.Called(ctx, filter, pq)
	if args.Get(0) == nil {
		return nil, args.Get(1).(pagination.Meta), args.Error(2)
	}
	return args.Get(0).([]usecase.DiscountListItem), args.Get(1).(pagination.Meta), args.Error(2)
}

func (m *MockDiscountUsecase) FindByID(ctx context.Context, id uint) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.DiscountResult), args.Error(1)
}

func (m *MockDiscountUsecase) FindByCode(ctx context.Context, code string) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.DiscountResult), args.Error(1)
}

func (m *MockDiscountUsecase) Update(ctx context.Context, id uint, input usecase.UpdateDiscountInput) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.DiscountResult), args.Error(1)
}

func (m *MockDiscountUsecase) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockDiscountUsecase) ValidateAndCalculate(ctx context.Context, code string, items []usecase.CartItemInput) (*usecase.CheckDiscountResult, error) {
	args := m.Called(ctx, code, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CheckDiscountResult), args.Error(1)
}

func setupDiscountApp(mockUC *MockDiscountUsecase, allowAuth bool, allowAdmin bool) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	dummyAuthMiddleware := func(c fiber.Ctx) error {
		if !allowAuth {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": 401, "message": "unauthorized"})
		}
		c.Locals("user_id", uint(1))
		return c.Next()
	}

	dummyAdminMiddleware := func(c fiber.Ctx) error {
		if !allowAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": 403, "message": "forbidden"})
		}
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewDiscountHandler(api, mockUC, validate, dummyAuthMiddleware, dummyAdminMiddleware)
	return app
}

func TestDiscountHandler_FindAll_Success(t *testing.T) {
	mockUC := new(MockDiscountUsecase)
	app := setupDiscountApp(mockUC, true, true)

	expectedDiscounts := []usecase.DiscountListItem{
		{ID: 1, Code: "PROMOBESAR", Name: "Promo Besar", DiscountType: "FIXED", MinPurchase: 50000, IsActive: true},
	}
	expectedMeta := pagination.Meta{
		CurrentPage: 1,
		PerPage:     10,
		TotalItems:  1,
		TotalPages:  1,
	}

	mockUC.On("FindAll", mock.Anything, repository.DiscountFilter{}, pagination.Query{Page: 1, PerPage: 10}).
		Return(expectedDiscounts, expectedMeta, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discounts?page=1&per_page=10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                        `json:"code"`
		Data []usecase.DiscountListItem `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.Len(t, res.Data, 1)
	assert.Equal(t, "PROMOBESAR", res.Data[0].Code)
	mockUC.AssertExpectations(t)
}

func TestDiscountHandler_CheckDiscount_Success(t *testing.T) {
	mockUC := new(MockDiscountUsecase)
	app := setupDiscountApp(mockUC, true, false) // User biasa (authenticated)

	expectedResult := &usecase.CheckDiscountResult{
		Code:           "DISKONHEMAT10",
		Name:           "Diskon Hemat 10%",
		DiscountAmount: 15000,
		FinalAmount:    135000,
	}

	items := []usecase.CartItemInput{
		{BookID: 1, Quantity: 2},
	}

	mockUC.On("ValidateAndCalculate", mock.Anything, "DISKONHEMAT10", items).Return(expectedResult, nil)

	payload := `{"code":"DISKONHEMAT10","items":[{"book_id":1,"quantity":2}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discounts/check", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                         `json:"code"`
		Data usecase.CheckDiscountResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.Equal(t, 15000.0, res.Data.DiscountAmount)
	assert.Equal(t, 135000.0, res.Data.FinalAmount)
	mockUC.AssertExpectations(t)
}

func TestDiscountHandler_CheckDiscount_ExpiredOrExhausted(t *testing.T) {
	mockUC := new(MockDiscountUsecase)
	app := setupDiscountApp(mockUC, true, false)

	items := []usecase.CartItemInput{
		{BookID: 1, Quantity: 1},
	}

	mockUC.On("ValidateAndCalculate", mock.Anything, "EXPIREDCODE", items).
		Return(nil, entity.ErrDiscountExpired)

	payload := `{"code":"EXPIREDCODE","items":[{"book_id":1,"quantity":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discounts/check", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertExpectations(t)
}
