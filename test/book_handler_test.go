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
	assert.Equal(t, 0, res.Data[0].Sold)
	assert.Contains(t, string(body), `"sold":0`)
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

	origPrice := 150000
	discPct := 10
	expected := &usecase.BookDetailResult{
		ID:                 1,
		Title:              "Bumi Manusia",
		Slug:               "bumi-manusia",
		Price:              135000,
		OriginalPrice:      &origPrice,
		DiscountPercentage: &discPct,
		Language:           "Indonesia",
		Length:             20.0,
		Width:              13.5,
		Height:             2.0,
		Condition:          "100% Baru & Segel Penerbit",
		Stock:              25,
		CoverImage:         func() *string { s := "https://example.com/cover.jpg"; return &s }(),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	mockUC.On("Create", mock.Anything, mock.MatchedBy(func(input usecase.CreateBookInput) bool {
		return input.CategoryID == 1 &&
			input.AuthorID == 2 &&
			input.PublisherID == 3 &&
			input.Title == "Bumi Manusia" &&
			input.ISBN != nil && *input.ISBN == "9789799731234" &&
			input.Synopsis != nil && *input.Synopsis == "Sebuah novel karya Pramoedya Ananta Toer." &&
			input.Pages != nil && *input.Pages == 535 &&
			input.PublicationYear != nil && *input.PublicationYear == 2005 &&
			input.Stock == 25 &&
			input.CoverImage != nil && *input.CoverImage == "https://example.com/cover.jpg" &&
			input.Language == "Indonesia" &&
			input.Length == 20.0 &&
			input.Width == 13.5 &&
			input.Height == 2.0 &&
			input.Condition == "100% Baru & Segel Penerbit" &&
			input.OriginalPrice != nil && *input.OriginalPrice == 150000.0
	})).Return(expected, nil)

	payload := `{
		"category_id": 1,
		"author_id": 2,
		"publisher_id": 3,
		"title": "Bumi Manusia",
		"isbn": "9789799731234",
		"synopsis": "Sebuah novel karya Pramoedya Ananta Toer.",
		"price": 135000,
		"original_price": 150000,
		"weight_grams": 450,
		"pages": 535,
		"publication_year": 2005,
		"stock": 25,
		"cover_image": "https://example.com/cover.jpg",
		"language": "Indonesia",
		"length": 20.0,
		"width": 13.5,
		"height": 2.0,
		"condition": "100% Baru & Segel Penerbit"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/books", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestBookHandler_Create_AutoDiscountFromPercentageOnly(t *testing.T) {
	mockUC := new(MockBookUsecase)
	app := setupBookApp(mockUC, true, true)

	origPrice := 100000
	discPct := 20
	expected := &usecase.BookDetailResult{
		ID:                 2,
		Title:              "Bumi Manusia",
		Slug:               "bumi-manusia",
		Price:              80000,
		OriginalPrice:      &origPrice,
		DiscountPercentage: &discPct,
		Stock:              15,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	mockUC.On("Create", mock.Anything, mock.MatchedBy(func(input usecase.CreateBookInput) bool {
		return input.Price == 100000 &&
			input.OriginalPrice == nil &&
			input.DiscountPercentage != nil && *input.DiscountPercentage == 20 &&
			input.Stock == 15
	})).Return(expected, nil)

	payload := `{
		"category_id": 1,
		"author_id": 2,
		"publisher_id": 3,
		"title": "Bumi Manusia",
		"isbn": "9789799731234",
		"synopsis": "Sebuah novel karya Pramoedya Ananta Toer.",
		"price": 100000,
		"discount_percentage": 20,
		"weight_grams": 450,
		"pages": 535,
		"publication_year": 2005,
		"stock": 15,
		"cover_image": "https://example.com/cover.jpg",
		"language": "Indonesia",
		"length": 20.0,
		"width": 13.5,
		"height": 2.0,
		"condition": "100% Baru & Segel Penerbit"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/books", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestBookHandler_Create_ValidationError_MissingRequiredFields(t *testing.T) {
	mockUC := new(MockBookUsecase)
	app := setupBookApp(mockUC, true, true)

	// Missing stock, cover_image, pages, publication_year, synopsis, isbn
	payload := `{
		"category_id": 1,
		"author_id": 2,
		"publisher_id": 3,
		"title": "Bumi Manusia",
		"price": 135000,
		"weight_grams": 450,
		"language": "Indonesia",
		"length": 20.0,
		"width": 13.5,
		"height": 2.0,
		"condition": "100% Baru & Segel Penerbit"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/books", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
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
		"isbn": "9789799731234",
		"synopsis": "Sebuah novel karya Pramoedya Ananta Toer.",
		"price": 135000,
		"weight_grams": 450,
		"pages": 535,
		"publication_year": 2005,
		"stock": 10,
		"cover_image": "https://example.com/cover.jpg",
		"language": "Indonesia",
		"length": 20.0,
		"width": 13.5,
		"height": 2.0,
		"condition": "100% Baru & Segel Penerbit"
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
