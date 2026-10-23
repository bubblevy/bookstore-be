package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWishlistRepository struct {
	mock.Mock
}

func (m *MockWishlistRepository) FindByUserID(ctx context.Context, userID uint, page, perPage int) ([]entity.Wishlist, int64, error) {
	args := m.Called(ctx, userID, page, perPage)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.Wishlist), args.Get(1).(int64), args.Error(2)
}

func (m *MockWishlistRepository) FindByUserAndBook(ctx context.Context, userID, bookID uint) (*entity.Wishlist, error) {
	args := m.Called(ctx, userID, bookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Wishlist), args.Error(1)
}

func (m *MockWishlistRepository) FindByID(ctx context.Context, id uint) (*entity.Wishlist, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Wishlist), args.Error(1)
}

func (m *MockWishlistRepository) Add(ctx context.Context, wishlist *entity.Wishlist) error {
	return m.Called(ctx, wishlist).Error(0)
}

func (m *MockWishlistRepository) DeleteByUserAndBook(ctx context.Context, userID, bookID uint) error {
	return m.Called(ctx, userID, bookID).Error(0)
}

func (m *MockWishlistRepository) DeleteByID(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockWishlistRepository) Clear(ctx context.Context, userID uint) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockWishlistRepository) Exists(ctx context.Context, userID, bookID uint) (bool, error) {
	args := m.Called(ctx, userID, bookID)
	return args.Bool(0), args.Error(1)
}

func TestWishlistUsecase_GetWishlist_Success(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	now := time.Now()
	wishlistItems := []entity.Wishlist{
		{
			ID:        1,
			UserID:    1,
			BookID:    10,
			CreatedAt: now,
			Book: entity.Book{
				Title:       "Laskar Pelangi",
				Slug:        "laskar-pelangi",
				Price:       85000,
				WeightGrams: 300,
				Stock:       15,
				IsActive:    true,
				Category:    entity.Category{Name: "Fiction"},
				Author:      entity.Author{Name: "Andrea Hirata"},
				Publisher:   entity.Publisher{Name: "Bentang Pustaka"},
			},
		},
	}

	mockWishlistRepo.On("FindByUserID", ctx, uint(1), 1, 10).Return(wishlistItems, int64(1), nil)

	items, meta, err := uc.GetWishlist(ctx, 1, 1, 10)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, uint(1), items[0].ID)
	assert.Equal(t, "Laskar Pelangi", items[0].Book.Title)
	assert.Equal(t, "Fiction", items[0].Book.CategoryName)
	assert.Equal(t, "Andrea Hirata", items[0].Book.AuthorName)
	assert.Equal(t, int64(1), meta.TotalItems)
	mockWishlistRepo.AssertExpectations(t)
}

func TestWishlistUsecase_AddToWishlist_Success(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	book := &entity.Book{
		Title:    "Bumi Manusia",
		Stock:    10,
		IsActive: true,
	}

	mockBookRepo.On("FindByID", ctx, uint(10)).Return(book, nil)
	mockWishlistRepo.On("Exists", ctx, uint(1), uint(10)).Return(false, nil)
	mockWishlistRepo.On("Add", ctx, mock.AnythingOfType("*entity.Wishlist")).Return(nil)

	err := uc.AddToWishlist(ctx, 1, 10)
	assert.NoError(t, err)
	mockBookRepo.AssertExpectations(t)
	mockWishlistRepo.AssertExpectations(t)
}

func TestWishlistUsecase_AddToWishlist_BookNotFound(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	mockBookRepo.On("FindByID", ctx, uint(999)).Return(nil, nil)

	err := uc.AddToWishlist(ctx, 1, 999)
	assert.ErrorIs(t, err, entity.ErrBookNotFound)
}

func TestWishlistUsecase_AddToWishlist_BookNotActive(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	book := &entity.Book{
		Title:    "Buku Nonaktif",
		IsActive: false,
	}
	mockBookRepo.On("FindByID", ctx, uint(10)).Return(book, nil)

	err := uc.AddToWishlist(ctx, 1, 10)
	assert.ErrorIs(t, err, entity.ErrBookNotActiveInWishlist)
}

func TestWishlistUsecase_AddToWishlist_AlreadyExists(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	book := &entity.Book{
		Title:    "Bumi Manusia",
		IsActive: true,
	}
	mockBookRepo.On("FindByID", ctx, uint(10)).Return(book, nil)
	mockWishlistRepo.On("Exists", ctx, uint(1), uint(10)).Return(true, nil)

	err := uc.AddToWishlist(ctx, 1, 10)
	assert.ErrorIs(t, err, entity.ErrWishlistAlreadyExists)
}

func TestWishlistUsecase_RemoveFromWishlist_Success(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	mockWishlistRepo.On("DeleteByUserAndBook", ctx, uint(1), uint(10)).Return(nil)

	err := uc.RemoveFromWishlist(ctx, 1, 10)
	assert.NoError(t, err)
	mockWishlistRepo.AssertExpectations(t)
}

func TestWishlistUsecase_RemoveFromWishlist_NotFound(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	mockWishlistRepo.On("DeleteByUserAndBook", ctx, uint(1), uint(999)).Return(entity.ErrWishlistItemNotFound)

	err := uc.RemoveFromWishlist(ctx, 1, 999)
	assert.ErrorIs(t, err, entity.ErrWishlistItemNotFound)
}

func TestWishlistUsecase_ClearWishlist_Success(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	mockWishlistRepo.On("Clear", ctx, uint(1)).Return(nil)

	err := uc.ClearWishlist(ctx, 1)
	assert.NoError(t, err)
	mockWishlistRepo.AssertExpectations(t)
}

func TestWishlistUsecase_CheckWishlist_Success(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	mockWishlistRepo.On("Exists", ctx, uint(1), uint(10)).Return(true, nil)

	exists, err := uc.CheckWishlist(ctx, 1, 10)
	assert.NoError(t, err)
	assert.True(t, exists)
	mockWishlistRepo.AssertExpectations(t)
}

func TestWishlistUsecase_MoveToCart_Success(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	book := &entity.Book{
		Title:    "Bumi Manusia",
		Stock:    5,
		IsActive: true,
	}

	mockWishlistRepo.On("Exists", ctx, uint(1), uint(10)).Return(true, nil)
	mockBookRepo.On("FindByID", ctx, uint(10)).Return(book, nil)
	mockCartUsecase.On("AddToCart", ctx, uint(1), usecase.AddToCartInput{BookID: 10, Quantity: 2}).Return(nil)
	mockWishlistRepo.On("DeleteByUserAndBook", ctx, uint(1), uint(10)).Return(nil)

	err := uc.MoveToCart(ctx, 1, 10, 2)
	assert.NoError(t, err)
	mockWishlistRepo.AssertExpectations(t)
	mockBookRepo.AssertExpectations(t)
	mockCartUsecase.AssertExpectations(t)
}

func TestWishlistUsecase_MoveToCart_InsufficientStock(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	book := &entity.Book{
		Title:    "Bumi Manusia",
		Stock:    2,
		IsActive: true,
	}

	mockWishlistRepo.On("Exists", ctx, uint(1), uint(10)).Return(true, nil)
	mockBookRepo.On("FindByID", ctx, uint(10)).Return(book, nil)

	err := uc.MoveToCart(ctx, 1, 10, 5)
	assert.ErrorIs(t, err, entity.ErrInsufficientStock)
}

func TestWishlistUsecase_MoveToCart_NotFoundInWishlist(t *testing.T) {
	mockWishlistRepo := new(MockWishlistRepository)
	mockBookRepo := new(MockBookRepository)
	mockCartUsecase := new(MockCartUsecase)
	uc := usecase.NewWishlistUsecase(mockWishlistRepo, mockBookRepo, mockCartUsecase)
	ctx := context.Background()

	mockWishlistRepo.On("Exists", ctx, uint(1), uint(10)).Return(false, nil)

	err := uc.MoveToCart(ctx, 1, 10, 1)
	assert.ErrorIs(t, err, entity.ErrWishlistItemNotFound)
}
