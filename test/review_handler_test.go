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
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockReviewUsecase struct {
	mock.Mock
}

func (m *MockReviewUsecase) GetBookReviews(ctx context.Context, slug string, filter repository.ReviewFilter, pq pagination.Query) (*dto.BookReviewsSummaryResponse, error) {
	args := m.Called(ctx, slug, filter, pq)
	if res := args.Get(0); res != nil {
		return res.(*dto.BookReviewsSummaryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewUsecase) CheckOrderReviewEligibility(ctx context.Context, userID uint, invoiceNumber string) (*dto.OrderReviewEligibilityResponse, error) {
	args := m.Called(ctx, userID, invoiceNumber)
	if res := args.Get(0); res != nil {
		return res.(*dto.OrderReviewEligibilityResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewUsecase) SubmitOrderReviews(ctx context.Context, userID uint, invoiceNumber string, req dto.SubmitOrderReviewsRequest) ([]dto.CustomerReviewItemResponse, error) {
	args := m.Called(ctx, userID, invoiceNumber, req)
	if res := args.Get(0); res != nil {
		return res.([]dto.CustomerReviewItemResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewUsecase) UpdateReview(ctx context.Context, userID uint, reviewID uint, req dto.UpdateReviewRequest) (*dto.CustomerReviewItemResponse, error) {
	args := m.Called(ctx, userID, reviewID, req)
	if res := args.Get(0); res != nil {
		return res.(*dto.CustomerReviewItemResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewUsecase) GetCustomerReviews(ctx context.Context, userID uint, pq pagination.Query) ([]dto.CustomerReviewItemResponse, pagination.Meta, error) {
	args := m.Called(ctx, userID, pq)
	if res := args.Get(0); res != nil {
		return res.([]dto.CustomerReviewItemResponse), args.Get(1).(pagination.Meta), args.Error(2)
	}
	return nil, pagination.Meta{}, args.Error(2)
}

func (m *MockReviewUsecase) AdminFindAllReviews(ctx context.Context, filter repository.ReviewFilter, pq pagination.Query) ([]dto.AdminReviewItemResponse, pagination.Meta, error) {
	args := m.Called(ctx, filter, pq)
	if res := args.Get(0); res != nil {
		return res.([]dto.AdminReviewItemResponse), args.Get(1).(pagination.Meta), args.Error(2)
	}
	return nil, pagination.Meta{}, args.Error(2)
}

func (m *MockReviewUsecase) AdminModerateReview(ctx context.Context, reviewID uint, req dto.AdminModerateReviewRequest) error {
	return m.Called(ctx, reviewID, req).Error(0)
}

func (m *MockReviewUsecase) AdminReplyReview(ctx context.Context, reviewID uint, req dto.AdminReplyReviewRequest) error {
	return m.Called(ctx, reviewID, req).Error(0)
}

func (m *MockReviewUsecase) AdminDeleteReview(ctx context.Context, reviewID uint) error {
	return m.Called(ctx, reviewID).Error(0)
}

func setupReviewApp(mockUC *MockReviewUsecase, isAuthenticated, isAdmin bool) *fiber.App {
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

	adminMiddleware := func(c fiber.Ctx) error {
		if !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    403,
				"status":  false,
				"message": "forbidden",
			})
		}
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewReviewHandler(api, mockUC, validate, authMiddleware, adminMiddleware)
	return app
}

func TestReviewHandler_GetBookReviews_Public(t *testing.T) {
	mockUC := new(MockReviewUsecase)
	app := setupReviewApp(mockUC, false, false)

	expectedResponse := &dto.BookReviewsSummaryResponse{
		Breakdown: dto.BookReviewBreakdownResponse{
			AverageRating: 4.9,
			TotalReviews:  20,
			Star5Count:    18,
			Star4Count:    2,
		},
		Reviews: []dto.BookReviewItemResponse{
			{ID: 1, Rating: 5, UserName: "J***e"},
		},
	}

	mockUC.On("GetBookReviews", mock.Anything, "belajar-golang", mock.Anything, mock.Anything).Return(expectedResponse, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books/belajar-golang/reviews?page=1&per_page=10", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestReviewHandler_CheckOrderEligibility(t *testing.T) {
	mockUC := new(MockReviewUsecase)
	app := setupReviewApp(mockUC, true, false)

	expected := &dto.OrderReviewEligibilityResponse{
		OrderID:       1,
		InvoiceNumber: "INV-001",
		IsEligible:    true,
	}

	mockUC.On("CheckOrderReviewEligibility", mock.Anything, uint(1), "INV-001").Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/INV-001/review-eligibility", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestReviewHandler_SubmitOrderReviews(t *testing.T) {
	mockUC := new(MockReviewUsecase)
	app := setupReviewApp(mockUC, true, false)

	submitReq := dto.SubmitOrderReviewsRequest{
		Reviews: []dto.CreateReviewItemRequest{
			{
				BookID:  10,
				Rating:  5,
				Comment: "Sangat bagus",
			},
		},
	}

	expectedResult := []dto.CustomerReviewItemResponse{
		{
			ID:        1,
			BookID:    10,
			Rating:    5,
			CreatedAt: time.Now(),
		},
	}

	mockUC.On("SubmitOrderReviews", mock.Anything, uint(1), "INV-001", mock.Anything).Return(expectedResult, nil)

	bodyBytes, _ := json.Marshal(submitReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/INV-001/reviews", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestReviewHandler_AdminEndpoints(t *testing.T) {
	mockUC := new(MockReviewUsecase)
	app := setupReviewApp(mockUC, true, true)

	t.Run("Admin moderate review", func(t *testing.T) {
		mockUC.On("AdminModerateReview", mock.Anything, uint(5), mock.Anything).Return(nil).Once()

		body := map[string]interface{}{
			"is_hidden": true,
			"reason":    "Spam review",
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/reviews/5/visibility", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Admin reply to review", func(t *testing.T) {
		mockUC.On("AdminReplyReview", mock.Anything, uint(5), mock.Anything).Return(nil).Once()

		body := map[string]interface{}{
			"reply": "Terima kasih atas pesanannya kak!",
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reviews/5/reply", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Admin delete review", func(t *testing.T) {
		mockUC.On("AdminDeleteReview", mock.Anything, uint(5)).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/reviews/5", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
