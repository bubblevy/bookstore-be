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
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBookUsecase struct {
	mock.Mock
}

func (m *MockBookUsecase) Create(ctx context.Context, input usecase.CreateBookInput) (*usecase.BookDetailResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.BookDetailResult), args.Error(1)
}

func (m *MockBookUsecase) FindAll(ctx context.Context, filter repository.BookFilter, pq pagination.Query) ([]usecase.BookListItem, pagination.Meta, error) {
	args := m.Called(ctx, filter, pq)
	if args.Get(0) == nil {
		return nil, args.Get(1).(pagination.Meta), args.Error(2)
	}
	return args.Get(0).([]usecase.BookListItem), args.Get(1).(pagination.Meta), args.Error(2)
}

func (m *MockBookUsecase) FindBySlug(ctx context.Context, slug string) (*usecase.BookDetailResult, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.BookDetailResult), args.Error(1)
}

func (m *MockBookUsecase) Update(ctx context.Context, id uint, input usecase.UpdateBookInput) (*usecase.BookDetailResult, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.BookDetailResult), args.Error(1)
}

func (m *MockBookUsecase) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func setupBookApp(mockUC *MockBookUsecase, allowAuth bool, allowAdmin bool) *fiber.App {
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
		c.Locals("is_admin", true)
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewBookHandler(api, mockUC, validate, dummyAuthMiddleware, dummyAdminMiddleware)
	return app
}

func TestBookHandler_FindAll_PublicAlwaysActive(t *testing.T) {
	mockUC := new(MockBookUsecase)
	app := setupBookApp(mockUC, false, false)

	expectedItems := []usecase.BookListItem{
		{ID: 1, Title: "Bumi", Slug: "bumi", Price: 95000, IsActive: true},
	}
	expectedMeta := pagination.Meta{CurrentPage: 1, PerPage: 10, TotalItems: 1, TotalPages: 1}

	activeTrue := true
	expectedFilter := repository.BookFilter{
		Search:        "bumi",
		CategorySlug:  "fiksi",
		AuthorSlug:    "",
		PublisherSlug: "",
		SortBy:        "newest",
		IsActive:      &activeTrue, // Memastikan pengunjung publik dikunci ke active = true
	}

	mockUC.On("FindAll", mock.Anything, expectedFilter, pagination.Query{Page: 1, PerPage: 10}).
		Return(expectedItems, expectedMeta, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books?search=bumi&category=fiksi&is_active=false&page=1&per_page=10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Status bool                   `json:"status"`
		Data   []usecase.BookListItem `json:"data"`
	}
	_ = json.Unmarshal(body, &res)
	assert.True(t, res.Status)
	assert.Len(t, res.Data, 1)
	mockUC.AssertExpectations(t)
}

func TestBookHandler_FindBySlug_HideInactiveFromPublic(t *testing.T) {
	mockUC := new(MockBookUsecase)
	app := setupBookApp(mockUC, false, false)

	inactiveBook := &usecase.BookDetailResult{
		ID:        1,
		Title:     "Draft Book",
		Slug:      "draft-book",
		IsActive:  false, // Nonaktif
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUC.On("FindBySlug", mock.Anything, "draft-book").Return(inactiveBook, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books/draft-book", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestBookHandler_Create_Success(t *testing.T) {
	mockUC := new(MockBookUsecase)
	app := setupBookApp(mockUC, true, true)

	expected := &usecase.BookDetailResult{
		ID:        1,
		Title:     "Bumi Manusia",
		Slug:      "bumi-manusia",
		Price:     135000,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUC.On("Create", mock.Anything, mock.MatchedBy(func(input usecase.CreateBookInput) bool {
		return input.CategoryID == 1 && input.AuthorID == 2 && input.PublisherID == 3 && input.Title == "Bumi Manusia"
	})).Return(expected, nil)

	payload := `{
		"category_id": 1,
		"author_id": 2,
		"publisher_id": 3,
		"title": "Bumi Manusia",
		"price": 135000,
		"weight_grams": 450
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/books", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestBookHandler_Create_SlugConflict(t *testing.T) {
	mockUC := new(MockBookUsecase)
	app := setupBookApp(mockUC, true, true)

	mockUC.On("Create", mock.Anything, mock.Anything).Return(nil, entity.ErrBookSlugConflict)

	payload := `{
		"category_id": 1,
		"author_id": 2,
		"publisher_id": 3,
		"title": "Bumi Manusia",
		"price": 135000,
		"weight_grams": 450
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/books", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestBookHandler_Delete_Success(t *testing.T) {
	mockUC := new(MockBookUsecase)
	app := setupBookApp(mockUC, true, true)

	mockUC.On("Delete", mock.Anything, uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/books/1", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}
