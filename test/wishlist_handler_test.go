package test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWishlistUsecase struct {
	mock.Mock
}

func (m *MockWishlistUsecase) GetWishlist(ctx context.Context, userID uint, page, perPage int) ([]usecase.WishlistItemResult, pagination.Meta, error) {
	args := m.Called(ctx, userID, page, perPage)
	if args.Get(0) == nil {
		return nil, args.Get(1).(pagination.Meta), args.Error(2)
	}
	return args.Get(0).([]usecase.WishlistItemResult), args.Get(1).(pagination.Meta), args.Error(2)
}

func (m *MockWishlistUsecase) AddToWishlist(ctx context.Context, userID uint, bookID uint) error {
	return m.Called(ctx, userID, bookID).Error(0)
}

func (m *MockWishlistUsecase) RemoveFromWishlist(ctx context.Context, userID uint, bookID uint) error {
	return m.Called(ctx, userID, bookID).Error(0)
}

func (m *MockWishlistUsecase) ClearWishlist(ctx context.Context, userID uint) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockWishlistUsecase) CheckWishlist(ctx context.Context, userID uint, bookID uint) (bool, error) {
	args := m.Called(ctx, userID, bookID)
	return args.Bool(0), args.Error(1)
}

func (m *MockWishlistUsecase) MoveToCart(ctx context.Context, userID uint, bookID uint, quantity int) error {
	return m.Called(ctx, userID, bookID, quantity).Error(0)
}

func setupWishlistApp(mockUC *MockWishlistUsecase, isAuthenticated bool) *fiber.App {
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
	handler.NewWishlistHandler(api, mockUC, validate, authMiddleware)
	return app
}

func TestWishlistHandler_GetWishlist_Success(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	items := []usecase.WishlistItemResult{
		{
			ID:        1,
			BookID:    10,
			CreatedAt: time.Now(),
			Book: usecase.WishlistBookResult{
				ID:           10,
				Title:        "Laskar Pelangi",
				Slug:         "laskar-pelangi",
				Price:        85000,
				Stock:        10,
				IsActive:     true,
				CategoryName: "Fiction",
			},
		},
	}
	meta := pagination.BuildMeta(1, 1, 10)

	mockUC.On("GetWishlist", mock.Anything, uint(1), 1, 10).Return(items, meta, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wishlist?page=1&per_page=10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int `json:"code"`
		Data []struct {
			ID   uint `json:"id"`
			Book struct {
				Title string `json:"title"`
			} `json:"book"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.Len(t, res.Data, 1)
	assert.Equal(t, "Laskar Pelangi", res.Data[0].Book.Title)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_CheckWishlist_Success(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	mockUC.On("CheckWishlist", mock.Anything, uint(1), uint(10)).Return(true, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wishlist/check/10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int `json:"code"`
		Data struct {
			BookID       uint `json:"book_id"`
			IsWishlisted bool `json:"is_wishlisted"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.True(t, res.Data.IsWishlisted)
	assert.Equal(t, uint(10), res.Data.BookID)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_AddItem_Success(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	mockUC.On("AddToWishlist", mock.Anything, uint(1), uint(10)).Return(nil)

	payload := `{"book_id": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlist/items", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_AddItem_AlreadyExists(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	mockUC.On("AddToWishlist", mock.Anything, uint(1), uint(10)).Return(entity.ErrWishlistAlreadyExists)

	payload := `{"book_id": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlist/items", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_DeleteItem_Success(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	mockUC.On("RemoveFromWishlist", mock.Anything, uint(1), uint(10)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/wishlist/items/10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_DeleteItem_NotFound(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	mockUC.On("RemoveFromWishlist", mock.Anything, uint(1), uint(999)).Return(entity.ErrWishlistItemNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/wishlist/items/999", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_MoveToCart_Success(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	mockUC.On("MoveToCart", mock.Anything, uint(1), uint(10), 1).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlist/items/10/move-to-cart", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_ClearWishlist_Success(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, true)

	mockUC.On("ClearWishlist", mock.Anything, uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/wishlist", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestWishlistHandler_Unauthorized(t *testing.T) {
	mockUC := new(MockWishlistUsecase)
	app := setupWishlistApp(mockUC, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wishlist", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
