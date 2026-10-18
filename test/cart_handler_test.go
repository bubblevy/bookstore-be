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
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCartUsecase struct {
	mock.Mock
}

func (m *MockCartUsecase) GetCart(ctx context.Context, userID uint) (*usecase.CartResult, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CartResult), args.Error(1)
}

func (m *MockCartUsecase) GetSelectedItems(ctx context.Context, userID uint, input usecase.GetSelectedCartItemsInput) (*usecase.SelectedCartItemsResult, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.SelectedCartItemsResult), args.Error(1)
}

func (m *MockCartUsecase) AddToCart(ctx context.Context, userID uint, input usecase.AddToCartInput) error {
	return m.Called(ctx, userID, input).Error(0)
}

func (m *MockCartUsecase) UpdateItem(ctx context.Context, userID uint, itemID uint, input usecase.UpdateCartItemInput) error {
	return m.Called(ctx, userID, itemID, input).Error(0)
}

func (m *MockCartUsecase) DeleteItem(ctx context.Context, userID uint, itemID uint) error {
	return m.Called(ctx, userID, itemID).Error(0)
}

func (m *MockCartUsecase) ClearCart(ctx context.Context, userID uint) error {
	return m.Called(ctx, userID).Error(0)
}

func setupCartApp(mockUC *MockCartUsecase, isAuthenticated bool) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	authMiddleware := func(c fiber.Ctx) error {
		if !isAuthenticated {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    401,
				"status":  false,
				"message": "unauthorized",
			})
		}
		c.Locals("user_id", uint(1))
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewCartHandler(api, mockUC, validate, authMiddleware)
	return app
}

func TestCartHandler_GetCart_Success(t *testing.T) {
	mockUC := new(MockCartUsecase)
	app := setupCartApp(mockUC, true)

	cartResult := &usecase.CartResult{
		ID:               1,
		TotalItems:       1,
		TotalQuantity:    2,
		TotalWeightGrams: 900,
		TotalPrice:       270000,
	}

	mockUC.On("GetCart", mock.Anything, uint(1)).Return(cartResult, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                `json:"code"`
		Data usecase.CartResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.Equal(t, 270000, res.Data.TotalPrice)
	mockUC.AssertExpectations(t)
}

func TestCartHandler_AddItem_Success(t *testing.T) {
	mockUC := new(MockCartUsecase)
	app := setupCartApp(mockUC, true)

	input := usecase.AddToCartInput{BookID: 1, Quantity: 2}
	mockUC.On("AddToCart", mock.Anything, uint(1), input).Return(nil)

	payload := `{"book_id": 1, "quantity": 2}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCartHandler_AddItem_InsufficientStock(t *testing.T) {
	mockUC := new(MockCartUsecase)
	app := setupCartApp(mockUC, true)

	input := usecase.AddToCartInput{BookID: 1, Quantity: 100}
	mockUC.On("AddToCart", mock.Anything, uint(1), input).Return(entity.ErrInsufficientStock)

	payload := `{"book_id": 1, "quantity": 100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCartHandler_GetSelectedItems_Success(t *testing.T) {
	mockUC := new(MockCartUsecase)
	app := setupCartApp(mockUC, true)

	input := usecase.GetSelectedCartItemsInput{ItemIDs: []uint{10, 12}}
	selectedResult := &usecase.SelectedCartItemsResult{
		TotalItems:       2,
		TotalQuantity:    3,
		TotalWeightGrams: 1350,
		TotalPrice:       365000,
	}

	mockUC.On("GetSelectedItems", mock.Anything, uint(1), input).Return(selectedResult, nil)

	payload := `{"item_ids": [10, 12]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/selected", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                             `json:"code"`
		Data usecase.SelectedCartItemsResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.Equal(t, 365000, res.Data.TotalPrice)
	mockUC.AssertExpectations(t)
}

func TestCartHandler_UpdateItem_NotFound(t *testing.T) {
	mockUC := new(MockCartUsecase)
	app := setupCartApp(mockUC, true)

	input := usecase.UpdateCartItemInput{Quantity: 5}
	mockUC.On("UpdateItem", mock.Anything, uint(1), uint(99), input).Return(entity.ErrCartItemNotFound)

	payload := `{"quantity": 5}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cart/items/99", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCartHandler_DeleteItem_Success(t *testing.T) {
	mockUC := new(MockCartUsecase)
	app := setupCartApp(mockUC, true)

	mockUC.On("DeleteItem", mock.Anything, uint(1), uint(10)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/items/10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCartHandler_ClearCart_Success(t *testing.T) {
	mockUC := new(MockCartUsecase)
	app := setupCartApp(mockUC, true)

	mockUC.On("ClearCart", mock.Anything, uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}
