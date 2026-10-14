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
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCK CATEGORY USECASE
// ============================================================================

type MockCategoryUsecase struct {
	mock.Mock
}

func (m *MockCategoryUsecase) Create(ctx context.Context, req usecase.CreateCategoryInput) (*usecase.CategoryResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CategoryResult), args.Error(1)
}

func (m *MockCategoryUsecase) FindAll(ctx context.Context) ([]usecase.CategoryList, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]usecase.CategoryList), args.Error(1)
}

func (m *MockCategoryUsecase) FindBySlug(ctx context.Context, slug string) (*usecase.CategoryResult, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CategoryResult), args.Error(1)
}

// Parameter: ctx, id uint, req usecase.UpdateCategoryInput
func (m *MockCategoryUsecase) Update(ctx context.Context, id uint, req usecase.UpdateCategoryInput) (*usecase.CategoryResult, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CategoryResult), args.Error(1)
}

func (m *MockCategoryUsecase) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ============================================================================
// TEST SETUP HELPER
// ============================================================================

func setupCategoryApp(mockUC *MockCategoryUsecase, allowAuth bool, allowAdmin bool) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	// Mock Auth Middleware
	dummyAuthMiddleware := func(c fiber.Ctx) error {
		if !allowAuth {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    401,
				"status":  false,
				"message": "unauthorized",
			})
		}
		c.Locals("user_id", uint(1))
		return c.Next()
	}

	// Mock Admin Role Middleware
	dummyAdminMiddleware := func(c fiber.Ctx) error {
		if !allowAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    403,
				"status":  false,
				"message": "forbidden: admin role required",
			})
		}
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewCategoryHandler(api, mockUC, validate, dummyAuthMiddleware, dummyAdminMiddleware)

	return app
}

// ============================================================================
// UNIT TESTS
// ============================================================================

func TestCategoryHandler_FindAll_Success(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	// Gunakan CategoryList, bukan CategoryResult
	expected := []usecase.CategoryList{
		{ID: 1, Name: "Fiksi", Slug: "fiksi"},
		{ID: 2, Name: "Sains", Slug: "sains"},
	}

	mockUC.On("FindAll", mock.Anything).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code   int                    `json:"code"`
		Status bool                   `json:"status"`
		Data   []usecase.CategoryList `json:"data"`
	}
	_ = json.Unmarshal(body, &res)

	assert.Equal(t, 200, res.Code)
	assert.True(t, res.Status)
	assert.Len(t, res.Data, 2)
	assert.Equal(t, "Fiksi", res.Data[0].Name)
	mockUC.AssertExpectations(t)
}

func TestCategoryHandler_FindBySlug_Success(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	expected := &usecase.CategoryResult{
		ID:        1,
		Name:      "Fiksi Populer",
		Slug:      "fiksi-populer",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUC.On("FindBySlug", mock.Anything, "fiksi-populer").Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories/fiksi-populer", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code   int                    `json:"code"`
		Status bool                   `json:"status"`
		Data   usecase.CategoryResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)

	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "fiksi-populer", res.Data.Slug)
	mockUC.AssertExpectations(t)
}

func TestCategoryHandler_FindBySlug_NotFound(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	mockUC.On("FindBySlug", mock.Anything, "unknown-slug").Return(nil, entity.ErrCategoryNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories/unknown-slug", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCategoryHandler_Create_Success(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	input := usecase.CreateCategoryInput{Name: "Teknologi"}
	expected := &usecase.CategoryResult{
		ID:        3,
		Name:      "Teknologi",
		Slug:      "teknologi",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUC.On("Create", mock.Anything, input).Return(expected, nil)

	reqPayload := `{"name": "Teknologi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(reqPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                    `json:"code"`
		Data usecase.CategoryResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.Equal(t, 201, res.Code)
	assert.Equal(t, "teknologi", res.Data.Slug)
	mockUC.AssertExpectations(t)
}

func TestCategoryHandler_Create_ValidationError(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	// Nama kosong / terlalu pendek (< 2 karakter)
	reqPayload := `{"name": "A"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(reqPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCategoryHandler_Create_SlugConflict(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	input := usecase.CreateCategoryInput{Name: "Teknologi"}
	mockUC.On("Create", mock.Anything, input).Return(nil, entity.ErrCategorySlugConflict)

	reqPayload := `{"name": "Teknologi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(reqPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCategoryHandler_Update_Success(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	input := usecase.UpdateCategoryInput{ID: 1, Name: "Fiksi Ilmiah"}
	expected := &usecase.CategoryResult{
		ID:        1,
		Name:      "Fiksi Ilmiah",
		Slug:      "fiksi-ilmiah",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Signature method menerima: (ctx, uint(1), input)
	mockUC.On("Update", mock.Anything, uint(1), input).Return(expected, nil)

	reqPayload := `{"name": "Fiksi Ilmiah"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/categories/1", strings.NewReader(reqPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCategoryHandler_Delete_Success(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	app := setupCategoryApp(mockUC, true, true)

	mockUC.On("Delete", mock.Anything, uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestCategoryHandler_Protected_Unauthorized(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	// allowAuth = false
	app := setupCategoryApp(mockUC, false, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"Bisnis"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCategoryHandler_Protected_ForbiddenNonAdmin(t *testing.T) {
	mockUC := new(MockCategoryUsecase)
	// allowAuth = true, allowAdmin = false
	app := setupCategoryApp(mockUC, true, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"Bisnis"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}
