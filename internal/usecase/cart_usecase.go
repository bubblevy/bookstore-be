package usecase

import (
	"context"
	"fmt"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
)

// Input Structs
type AddToCartInput struct {
	BookID   uint
	Quantity int
}

type UpdateCartItemInput struct {
	Quantity int
}

// Result Structs
type CartBookSummaryResult struct {
	ID          uint    `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Price       float64 `json:"price"`
	WeightGrams int     `json:"weight_grams"`
	Stock       int     `json:"stock"`
	CoverImage  *string `json:"cover_image,omitempty"`
	IsActive    bool    `json:"is_active"`
}

type CartItemResult struct {
	ID             uint                  `json:"id"`
	BookID         uint                  `json:"book_id"`
	Quantity       int                   `json:"quantity"`
	Subtotal       float64               `json:"subtotal"`
	SubweightGrams int                   `json:"subweight_grams"`
	Book           CartBookSummaryResult `json:"book"`
}

type CartResult struct {
	ID               uint             `json:"id"`
	TotalItems       int              `json:"total_items"`
	TotalQuantity    int              `json:"total_quantity"`
	TotalWeightGrams int              `json:"total_weight_grams"`
	TotalPrice       float64          `json:"total_price"`
	Items            []CartItemResult `json:"items"`
}

type GetSelectedCartItemsInput struct {
	ItemIDs []uint
}

type SelectedCartItemsResult struct {
	TotalItems       int              `json:"total_items"`
	TotalQuantity    int              `json:"total_quantity"`
	TotalWeightGrams int              `json:"total_weight_grams"`
	TotalPrice       float64          `json:"total_price"`
	Items            []CartItemResult `json:"items"`
}

type CartUsecase interface {
	GetCart(ctx context.Context, userID uint) (*CartResult, error)
	GetSelectedItems(ctx context.Context, userID uint, input GetSelectedCartItemsInput) (*SelectedCartItemsResult, error)
	AddToCart(ctx context.Context, userID uint, input AddToCartInput) error
	UpdateItem(ctx context.Context, userID uint, itemID uint, input UpdateCartItemInput) error
	DeleteItem(ctx context.Context, userID uint, itemID uint) error
	ClearCart(ctx context.Context, userID uint) error
}

type cartUsecase struct {
	cartRepo repository.CartRepository
	bookRepo repository.BookRepository
}

func NewCartUsecase(cartRepo repository.CartRepository, bookRepo repository.BookRepository) CartUsecase {
	return &cartUsecase{
		cartRepo: cartRepo,
		bookRepo: bookRepo,
	}
}

func (u *cartUsecase) GetCart(ctx context.Context, userID uint) (*CartResult, error) {
	cart, err := u.cartRepo.FindCartWithItems(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed fetching cart: %w", err)
	}

	var totalQty int
	var totalWeight int
	var totalPrice float64

	items := make([]CartItemResult, 0, len(cart.Items))
	for _, it := range cart.Items {
		subtotal := it.Book.Price * float64(it.Quantity)
		subweight := it.Book.WeightGrams * it.Quantity

		totalQty += it.Quantity
		totalWeight += subweight
		totalPrice += subtotal

		items = append(items, CartItemResult{
			ID:             it.ID,
			BookID:         it.BookID,
			Quantity:       it.Quantity,
			Subtotal:       subtotal,
			SubweightGrams: subweight,
			Book: CartBookSummaryResult{
				ID:          it.Book.ID,
				Title:       it.Book.Title,
				Slug:        it.Book.Slug,
				Price:       it.Book.Price,
				WeightGrams: it.Book.WeightGrams,
				Stock:       it.Book.Stock,
				CoverImage:  it.Book.CoverImage,
				IsActive:    it.Book.IsActive,
			},
		})
	}

	return &CartResult{
		ID:               cart.ID,
		TotalItems:       len(items),
		TotalQuantity:    totalQty,
		TotalWeightGrams: totalWeight,
		TotalPrice:       totalPrice,
		Items:            items,
	}, nil
}

func (u *cartUsecase) GetSelectedItems(ctx context.Context, userID uint, input GetSelectedCartItemsInput) (*SelectedCartItemsResult, error) {
	if len(input.ItemIDs) == 0 {
		return nil, entity.ErrCartEmpty
	}

	cart, err := u.cartRepo.GetOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed fetching cart: %w", err)
	}

	items, err := u.cartRepo.FindSelectedItems(ctx, cart.ID, input.ItemIDs)
	if err != nil {
		return nil, fmt.Errorf("failed fetching selected items: %w", err)
	}

	if len(items) == 0 {
		return nil, entity.ErrCartItemNotFound
	}

	// Check if all requested item IDs are found in the cart
	if len(items) != len(input.ItemIDs) {
		return nil, entity.ErrCartItemNotFound
	}

	var totalQty int
	var totalWeight int
	var totalPrice float64

	resultItems := make([]CartItemResult, 0, len(items))
	for _, it := range items {
		if !it.Book.IsActive {
			return nil, fmt.Errorf("book %q is no longer active", it.Book.Title)
		}
		if it.Quantity > it.Book.Stock {
			return nil, fmt.Errorf("book %q exceeds available stock", it.Book.Title)
		}

		subtotal := it.Book.Price * float64(it.Quantity)
		subweight := it.Book.WeightGrams * it.Quantity

		totalQty += it.Quantity
		totalWeight += subweight
		totalPrice += subtotal

		resultItems = append(resultItems, CartItemResult{
			ID:             it.ID,
			BookID:         it.BookID,
			Quantity:       it.Quantity,
			Subtotal:       subtotal,
			SubweightGrams: subweight,
			Book: CartBookSummaryResult{
				ID:          it.Book.ID,
				Title:       it.Book.Title,
				Slug:        it.Book.Slug,
				Price:       it.Book.Price,
				WeightGrams: it.Book.WeightGrams,
				Stock:       it.Book.Stock,
				CoverImage:  it.Book.CoverImage,
				IsActive:    it.Book.IsActive,
			},
		})
	}

	return &SelectedCartItemsResult{
		TotalItems:       len(resultItems),
		TotalQuantity:    totalQty,
		TotalWeightGrams: totalWeight,
		TotalPrice:       totalPrice,
		Items:            resultItems,
	}, nil
}

func (u *cartUsecase) AddToCart(ctx context.Context, userID uint, input AddToCartInput) error {
	book, err := u.bookRepo.FindByID(ctx, input.BookID)
	if err != nil {
		return fmt.Errorf("failed fetching book: %w", err)
	}
	if book == nil || !book.IsActive {
		return entity.ErrBookNotActiveInCart
	}

	cart, err := u.cartRepo.GetOrCreateByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed getting cart: %w", err)
	}

	existingItem, err := u.cartRepo.FindItemByCartAndBook(ctx, cart.ID, input.BookID)
	if err != nil {
		return fmt.Errorf("failed checking existing item: %w", err)
	}

	targetQty := input.Quantity
	if existingItem != nil {
		targetQty += existingItem.Quantity
	}

	if targetQty > book.Stock {
		return entity.ErrInsufficientStock
	}

	item := &entity.CartItem{
		CartID:   cart.ID,
		BookID:   input.BookID,
		Quantity: input.Quantity,
	}

	return u.cartRepo.AddItem(ctx, item)
}

func (u *cartUsecase) UpdateItem(ctx context.Context, userID uint, itemID uint, input UpdateCartItemInput) error {
	cart, err := u.cartRepo.GetOrCreateByUserID(ctx, userID)
	if err != nil {
		return err
	}

	item, err := u.cartRepo.FindItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil || item.CartID != cart.ID {
		return entity.ErrCartItemNotFound
	}

	if input.Quantity > item.Book.Stock {
		return entity.ErrInsufficientStock
	}

	return u.cartRepo.UpdateItemQuantity(ctx, itemID, input.Quantity)
}

func (u *cartUsecase) DeleteItem(ctx context.Context, userID uint, itemID uint) error {
	cart, err := u.cartRepo.GetOrCreateByUserID(ctx, userID)
	if err != nil {
		return err
	}

	item, err := u.cartRepo.FindItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil || item.CartID != cart.ID {
		return entity.ErrCartItemNotFound
	}

	return u.cartRepo.DeleteItem(ctx, itemID)
}

func (u *cartUsecase) ClearCart(ctx context.Context, userID uint) error {
	cart, err := u.cartRepo.GetOrCreateByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return u.cartRepo.ClearCart(ctx, cart.ID)
}
