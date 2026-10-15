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

type MockAuthorUsecase struct {
	mock.Mock
}

func (m *MockAuthorUsecase) Create(ctx context.Context, req usecase.CreateAuthorInput) (*usecase.AuthorResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthorResult), args.Error(1)
}

func (m *MockAuthorUsecase) FindAll(ctx context.Context, pq pagination.Query) ([]usecase.AuthorList, pagination.Meta, error) {
	args := m.Called(ctx, pq)
	if args.Get(0) == nil {
		return nil, args.Get(1).(pagination.Meta), args.Error(2)
	}
	return args.Get(0).([]usecase.AuthorList), args.Get(1).(pagination.Meta), args.Error(2)
}

func (m *MockAuthorUsecase) FindBySlug(ctx context.Context, slug string, bookPq pagination.Query) (*usecase.AuthorDetailResult, error) {
	args := m.Called(ctx, slug, bookPq)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthorDetailResult), args.Error(1)
}

func (m *MockAuthorUsecase) Update(ctx context.Context, id uint, req usecase.UpdateAuthorInput) (*usecase.AuthorResult, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthorResult), args.Error(1)
}

func (m *MockAuthorUsecase) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupAuthorApp(mockUC *MockAuthorUsecase, allowAuth bool, allowAdmin bool) *fiber.App {
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
	handler.NewAuthorHandler(api, mockUC, validate, dummyAuthMiddleware, dummyAdminMiddleware)
	return app
}

func TestAuthorHandler_FindAll_Success(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, false, false)

	expectedAuthors := []usecase.AuthorList{
		{ID: 1, Name: "Tere Liye", Slug: "tere-liye"},
	}
	expectedMeta := pagination.Meta{
		CurrentPage: 1,
		PerPage:     10,
		TotalItems:  1,
		TotalPages:  1,
	}

	mockUC.On("FindAll", mock.Anything, pagination.Query{Page: 1, PerPage: 10}).
		Return(expectedAuthors, expectedMeta, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/authors?page=1&per_page=10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code    int                  `json:"code"`
		Status  bool                 `json:"status"`
		Message string               `json:"message"`
		Data    []usecase.AuthorList `json:"data"`
		Meta    pagination.Meta      `json:"meta"`
	}
	err = json.Unmarshal(body, &res)
	assert.NoError(t, err)
	assert.True(t, res.Status)
	assert.Len(t, res.Data, 1)
	assert.Equal(t, int64(1), res.Meta.TotalItems)
	mockUC.AssertExpectations(t)
}

func TestAuthorHandler_FindBySlug_Success(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, false, false)

	expectedDetail := &usecase.AuthorDetailResult{
		ID:    1,
		Name:  "Tere Liye",
		Slug:  "tere-liye",
		Books: []usecase.AuthorBookItem{},
		BooksMeta: pagination.Meta{
			CurrentPage: 1,
			PerPage:     10,
			TotalItems:  0,
			TotalPages:  0,
		},
	}

	mockUC.On("FindBySlug", mock.Anything, "tere-liye", pagination.Query{Page: 1, PerPage: 10}).
		Return(expectedDetail, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/authors/tere-liye", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthorHandler_FindBySlug_NotFound(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, false, false)

	mockUC.On("FindBySlug", mock.Anything, "unknown", pagination.Query{Page: 1, PerPage: 10}).
		Return(nil, entity.ErrAuthorNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/authors/unknown", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthorHandler_Create_Success(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, true, true)

	bio := "Penulis fiksi populer"
	input := usecase.CreateAuthorInput{Name: "Tere Liye", Bio: &bio}
	expected := &usecase.AuthorResult{
		ID:        1,
		Name:      "Tere Liye",
		Slug:      "tere-liye",
		Bio:       &bio,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUC.On("Create", mock.Anything, input).Return(expected, nil)

	payload := `{"name":"Tere Liye","bio":"Penulis fiksi populer"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/authors", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthorHandler_Create_ForbiddenNonAdmin(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, true, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authors", strings.NewReader(`{"name":"Tere Liye"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAuthorHandler_Create_SlugConflict(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, true, true)

	input := usecase.CreateAuthorInput{Name: "Tere Liye"}
	mockUC.On("Create", mock.Anything, input).Return(nil, entity.ErrAuthorSlugConflict)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authors", strings.NewReader(`{"name":"Tere Liye"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthorHandler_Update_Success(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, true, true)

	bio := "Bio updated"
	input := usecase.UpdateAuthorInput{Name: "Tere Liye", Bio: &bio}
	expected := &usecase.AuthorResult{
		ID:        1,
		Name:      "Tere Liye",
		Slug:      "tere-liye",
		Bio:       &bio,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUC.On("Update", mock.Anything, uint(1), input).Return(expected, nil)

	payload := `{"name":"Tere Liye","bio":"Bio updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/authors/1", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthorHandler_Delete_Success(t *testing.T) {
	mockUC := new(MockAuthorUsecase)
	app := setupAuthorApp(mockUC, true, true)

	mockUC.On("Delete", mock.Anything, uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/authors/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}
