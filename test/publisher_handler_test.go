package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPublisherUsecase struct {
	mock.Mock
}

func (m *MockPublisherUsecase) Create(ctx context.Context, req usecase.CreatePublisherInput) (*usecase.PublisherResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.PublisherResult), args.Error(1)
}

func (m *MockPublisherUsecase) FindAll(ctx context.Context) ([]usecase.PublisherList, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]usecase.PublisherList), args.Error(1)
}

func (m *MockPublisherUsecase) FindBySlug(ctx context.Context, slug string) (*usecase.PublisherResult, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.PublisherResult), args.Error(1)
}

func (m *MockPublisherUsecase) Update(ctx context.Context, id uint, req usecase.UpdatePublisherInput) (*usecase.PublisherResult, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.PublisherResult), args.Error(1)
}

func (m *MockPublisherUsecase) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupPublisherApp(mockUC *MockPublisherUsecase, allowAuth bool, allowAdmin bool) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	dummyAuthMiddleware := func(c fiber.Ctx) error {
		if !allowAuth {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
		}
		return c.Next()
	}

	dummyAdminMiddleware := func(c fiber.Ctx) error {
		if !allowAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "forbidden"})
		}
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewPublisherHandler(api, mockUC, validate, dummyAuthMiddleware, dummyAdminMiddleware)
	return app
}

func TestPublisherHandler_FindAll_Success(t *testing.T) {
	mockUC := new(MockPublisherUsecase)
	app := setupPublisherApp(mockUC, false, false)

	expected := []usecase.PublisherList{
		{ID: 1, Name: "Gramedia", Slug: "gramedia"},
	}
	mockUC.On("FindAll", mock.Anything).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishers", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestPublisherHandler_Create_Success(t *testing.T) {
	mockUC := new(MockPublisherUsecase)
	app := setupPublisherApp(mockUC, true, true)

	input := usecase.CreatePublisherInput{Name: "Mizan"}
	expected := &usecase.PublisherResult{
		ID:        2,
		Name:      "Mizan",
		Slug:      "mizan",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUC.On("Create", mock.Anything, input).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/publishers", strings.NewReader(`{"name":"Mizan"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestPublisherHandler_Create_ForbiddenNonAdmin(t *testing.T) {
	mockUC := new(MockPublisherUsecase)
	app := setupPublisherApp(mockUC, true, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/publishers", strings.NewReader(`{"name":"Mizan"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestPublisherHandler_Delete_Success(t *testing.T) {
	mockUC := new(MockPublisherUsecase)
	app := setupPublisherApp(mockUC, true, true)

	mockUC.On("Delete", mock.Anything, uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/publishers/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}
