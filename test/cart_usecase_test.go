package test

import (
	"context"
	"testing"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) GetOrCreateByUserID(ctx context.Context, userID uint) (*entity.Cart, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Cart), args.Error(1)
}

func (m *MockCartRepository) FindCartWithItems(ctx context.Context, userID uint) (*entity.Cart, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Cart), args.Error(1)
}

func (m *MockCartRepository) FindItemByID(ctx context.Context, itemID uint) (*entity.CartItem, error) {
	args := m.Called(ctx, itemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.CartItem), args.Error(1)
}

func (m *MockCartRepository) FindItemByCartAndBook(ctx context.Context, cartID, bookID uint) (*entity.CartItem, error) {
	args := m.Called(ctx, cartID, bookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.CartItem), args.Error(1)
}

func (m *MockCartRepository) FindSelectedItems(ctx context.Context, cartID uint, itemIDs []uint) ([]entity.CartItem, error) {
	args := m.Called(ctx, cartID, itemIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.CartItem), args.Error(1)
}

func (m *MockCartRepository) AddItem(ctx context.Context, item *entity.CartItem) error {
	return m.Called(ctx, item).Error(0)
}

func (m *MockCartRepository) UpdateItemQuantity(ctx context.Context, itemID uint, quantity int) error {
	return m.Called(ctx, itemID, quantity).Error(0)
}

func (m *MockCartRepository) DeleteItem(ctx context.Context, itemID uint) error {
	return m.Called(ctx, itemID).Error(0)
}

func (m *MockCartRepository) DeleteSelectedItems(ctx context.Context, cartID uint, itemIDs []uint) error {
	return m.Called(ctx, cartID, itemIDs).Error(0)
}

func (m *MockCartRepository) ClearCart(ctx context.Context, cartID uint) error {
	return m.Called(ctx, cartID).Error(0)
}

func TestCartUsecase_GetCart_Success(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewCartUsecase(mockCartRepo, mockBookRepo)
	ctx := context.Background()

	cart := &entity.Cart{
		ID:     1,
		UserID: 1,
		Items: []entity.CartItem{
			{
				ID:       10,
				CartID:   1,
				BookID:   1,
				Quantity: 2,
				Book: entity.Book{
					ID:          1,
					Title:       "Bumi Manusia",
					Price:       100000,
					WeightGrams: 400,
					Stock:       10,
					IsActive:    true,
				},
			},
			{
				ID:       11,
				CartID:   1,
				BookID:   2,
				Quantity: 1,
				Book: entity.Book{
					ID:          2,
					Title:       "Anak Semua Bangsa",
					Price:       80000,
					WeightGrams: 350,
					Stock:       5,
					IsActive:    true,
				},
			},
		},
	}

	mockCartRepo.On("FindCartWithItems", ctx, uint(1)).Return(cart, nil)

	res, err := uc.GetCart(ctx, 1)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, 2, res.TotalItems)
	assert.Equal(t, 3, res.TotalQuantity)
	assert.Equal(t, 1150, res.TotalWeightGrams) // (400*2) + (350*1)
	assert.Equal(t, 280000, res.TotalPrice)   // (100000*2) + (80000*1)
	mockCartRepo.AssertExpectations(t)
}

func TestCartUsecase_AddToCart_Success_NewItem(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewCartUsecase(mockCartRepo, mockBookRepo)
	ctx := context.Background()

	book := &entity.Book{ID: 1, Stock: 10, IsActive: true}
	cart := &entity.Cart{ID: 1, UserID: 1}

	mockBookRepo.On("FindByID", ctx, uint(1)).Return(book, nil)
	mockCartRepo.On("GetOrCreateByUserID", ctx, uint(1)).Return(cart, nil)
	mockCartRepo.On("FindItemByCartAndBook", ctx, uint(1), uint(1)).Return(nil, nil)
	mockCartRepo.On("AddItem", ctx, mock.MatchedBy(func(item *entity.CartItem) bool {
		return item.CartID == 1 && item.BookID == 1 && item.Quantity == 2
	})).Return(nil)

	err := uc.AddToCart(ctx, 1, usecase.AddToCartInput{BookID: 1, Quantity: 2})
	assert.NoError(t, err)
	mockBookRepo.AssertExpectations(t)
	mockCartRepo.AssertExpectations(t)
}

func TestCartUsecase_AddToCart_InsufficientStockAccumulation(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewCartUsecase(mockCartRepo, mockBookRepo)
	ctx := context.Background()

	book := &entity.Book{ID: 1, Stock: 5, IsActive: true}
	cart := &entity.Cart{ID: 1, UserID: 1}
	existingItem := &entity.CartItem{ID: 10, CartID: 1, BookID: 1, Quantity: 4}

	mockBookRepo.On("FindByID", ctx, uint(1)).Return(book, nil)
	mockCartRepo.On("GetOrCreateByUserID", ctx, uint(1)).Return(cart, nil)
	mockCartRepo.On("FindItemByCartAndBook", ctx, uint(1), uint(1)).Return(existingItem, nil)

	// Ingin tambah 2 padahal sudah ada 4 (Total 6 > Stock 5)
	err := uc.AddToCart(ctx, 1, usecase.AddToCartInput{BookID: 1, Quantity: 2})
	assert.ErrorIs(t, err, entity.ErrInsufficientStock)
}

func TestCartUsecase_GetSelectedItems_Success(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewCartUsecase(mockCartRepo, mockBookRepo)
	ctx := context.Background()

	cart := &entity.Cart{ID: 1, UserID: 1}
	selectedItems := []entity.CartItem{
		{
			ID:       10,
			CartID:   1,
			BookID:   1,
			Quantity: 2,
			Book: entity.Book{
				ID:          1,
				Title:       "Bumi Manusia",
				Price:       100000,
				WeightGrams: 400,
				Stock:       5,
				IsActive:    true,
			},
		},
	}

	mockCartRepo.On("GetOrCreateByUserID", ctx, uint(1)).Return(cart, nil)
	mockCartRepo.On("FindSelectedItems", ctx, uint(1), []uint{10}).Return(selectedItems, nil)

	res, err := uc.GetSelectedItems(ctx, 1, usecase.GetSelectedCartItemsInput{ItemIDs: []uint{10}})
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 1, res.TotalItems)
	assert.Equal(t, 2, res.TotalQuantity)
	assert.Equal(t, 800, res.TotalWeightGrams)
	assert.Equal(t, 200000, res.TotalPrice)
	mockCartRepo.AssertExpectations(t)
}

func TestCartUsecase_GetSelectedItems_ItemNotFoundOrMismatch(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewCartUsecase(mockCartRepo, mockBookRepo)
	ctx := context.Background()

	cart := &entity.Cart{ID: 1, UserID: 1}

	mockCartRepo.On("GetOrCreateByUserID", ctx, uint(1)).Return(cart, nil)
	// Request 2 item IDs (10 & 99), tapi hanya 10 yang ditemukan di cart user
	mockCartRepo.On("FindSelectedItems", ctx, uint(1), []uint{10, 99}).Return([]entity.CartItem{
		{ID: 10, CartID: 1, BookID: 1, Quantity: 1, Book: entity.Book{Stock: 10, IsActive: true}},
	}, nil)

	res, err := uc.GetSelectedItems(ctx, 1, usecase.GetSelectedCartItemsInput{ItemIDs: []uint{10, 99}})
	assert.ErrorIs(t, err, entity.ErrCartItemNotFound)
	assert.Nil(t, res)
}

func TestCartUsecase_UpdateItem_Success(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewCartUsecase(mockCartRepo, mockBookRepo)
	ctx := context.Background()

	cart := &entity.Cart{ID: 1, UserID: 1}
	item := &entity.CartItem{
		ID:       10,
		CartID:   1,
		BookID:   1,
		Quantity: 1,
		Book:     entity.Book{ID: 1, Stock: 5, IsActive: true},
	}

	mockCartRepo.On("GetOrCreateByUserID", ctx, uint(1)).Return(cart, nil)
	mockCartRepo.On("FindItemByID", ctx, uint(10)).Return(item, nil)
	mockCartRepo.On("UpdateItemQuantity", ctx, uint(10), 3).Return(nil)

	err := uc.UpdateItem(ctx, 1, 10, usecase.UpdateCartItemInput{Quantity: 3})
	assert.NoError(t, err)
	mockCartRepo.AssertExpectations(t)
}

func TestCartUsecase_DeleteItem_NotFoundOrDifferentOwner(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewCartUsecase(mockCartRepo, mockBookRepo)
	ctx := context.Background()

	cart := &entity.Cart{ID: 1, UserID: 1}
	itemOtherUser := &entity.CartItem{ID: 10, CartID: 999} // Milik cart lain

	mockCartRepo.On("GetOrCreateByUserID", ctx, uint(1)).Return(cart, nil)
	mockCartRepo.On("FindItemByID", ctx, uint(10)).Return(itemOtherUser, nil)

	err := uc.DeleteItem(ctx, 1, 10)
	assert.ErrorIs(t, err, entity.ErrCartItemNotFound)
}
