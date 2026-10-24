package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
)

type WishlistBookResult struct {
	ID                 uint    `json:"id"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Price              int     `json:"price"`
	OriginalPrice      *int    `json:"original_price,omitempty"`
	DiscountPercentage *int    `json:"discount_percentage,omitempty"`
	WeightGrams        int     `json:"weight_grams"`
	Stock              int     `json:"stock"`
	Sold               int     `json:"sold"`
	CoverImage         *string `json:"cover_image,omitempty"`
	IsActive           bool    `json:"is_active"`
	CategoryName       string  `json:"category_name,omitempty"`
	AuthorName         string  `json:"author_name,omitempty"`
	PublisherName      string  `json:"publisher_name,omitempty"`
}

type WishlistItemResult struct {
	ID        uint               `json:"id"`
	BookID    uint               `json:"book_id"`
	Book      WishlistBookResult `json:"book"`
	CreatedAt time.Time          `json:"created_at"`
}

type WishlistUsecase interface {
	GetWishlist(ctx context.Context, userID uint, page, perPage int) ([]WishlistItemResult, pagination.Meta, error)
	AddToWishlist(ctx context.Context, userID uint, bookID uint) error
	RemoveFromWishlist(ctx context.Context, userID uint, bookID uint) error
	ClearWishlist(ctx context.Context, userID uint) error
	CheckWishlist(ctx context.Context, userID uint, bookID uint) (bool, error)
	MoveToCart(ctx context.Context, userID uint, bookID uint, quantity int) error
}

type wishlistUsecase struct {
	wishlistRepo repository.WishlistRepository
	bookRepo     repository.BookRepository
	cartUsecase  CartUsecase
}

func NewWishlistUsecase(
	wishlistRepo repository.WishlistRepository,
	bookRepo repository.BookRepository,
	cartUsecase CartUsecase,
) WishlistUsecase {
	return &wishlistUsecase{
		wishlistRepo: wishlistRepo,
		bookRepo:     bookRepo,
		cartUsecase:  cartUsecase,
	}
}

func (u *wishlistUsecase) GetWishlist(ctx context.Context, userID uint, page, perPage int) ([]WishlistItemResult, pagination.Meta, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}

	items, total, err := u.wishlistRepo.FindByUserID(ctx, userID, page, perPage)
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed fetching wishlist: %w", err)
	}

	results := make([]WishlistItemResult, 0, len(items))
	for _, it := range items {
		results = append(results, WishlistItemResult{
			ID:     it.ID,
			BookID: it.BookID,
			Book: WishlistBookResult{
				ID:                 it.Book.ID,
				Title:              it.Book.Title,
				Slug:               it.Book.Slug,
				Price:              it.Book.Price,
				OriginalPrice:      it.Book.OriginalPrice,
				DiscountPercentage: it.Book.DiscountPercentage,
				WeightGrams:        it.Book.WeightGrams,
				Stock:              it.Book.Stock,
				Sold:               it.Book.Sold,
				CoverImage:         it.Book.CoverImage,
				IsActive:           it.Book.IsActive,
				CategoryName:       it.Book.Category.Name,
				AuthorName:         it.Book.Author.Name,
				PublisherName:      it.Book.Publisher.Name,
			},
			CreatedAt: it.CreatedAt,
		})
	}

	meta := pagination.BuildMeta(total, page, perPage)
	return results, meta, nil
}

func (u *wishlistUsecase) AddToWishlist(ctx context.Context, userID uint, bookID uint) error {
	book, err := u.bookRepo.FindByID(ctx, bookID)
	if err != nil {
		return fmt.Errorf("failed fetching book: %w", err)
	}
	if book == nil {
		return entity.ErrBookNotFound
	}
	if !book.IsActive {
		return entity.ErrBookNotActiveInWishlist
	}

	exists, err := u.wishlistRepo.Exists(ctx, userID, bookID)
	if err != nil {
		return fmt.Errorf("failed checking wishlist item: %w", err)
	}
	if exists {
		return entity.ErrWishlistAlreadyExists
	}

	item := &entity.Wishlist{
		UserID: userID,
		BookID: bookID,
	}

	if err := u.wishlistRepo.Add(ctx, item); err != nil {
		return fmt.Errorf("failed adding book to wishlist: %w", err)
	}

	return nil
}

func (u *wishlistUsecase) RemoveFromWishlist(ctx context.Context, userID uint, bookID uint) error {
	if err := u.wishlistRepo.DeleteByUserAndBook(ctx, userID, bookID); err != nil {
		return err
	}
	return nil
}

func (u *wishlistUsecase) ClearWishlist(ctx context.Context, userID uint) error {
	if err := u.wishlistRepo.Clear(ctx, userID); err != nil {
		return fmt.Errorf("failed clearing wishlist: %w", err)
	}
	return nil
}

func (u *wishlistUsecase) CheckWishlist(ctx context.Context, userID uint, bookID uint) (bool, error) {
	return u.wishlistRepo.Exists(ctx, userID, bookID)
}

func (u *wishlistUsecase) MoveToCart(ctx context.Context, userID uint, bookID uint, quantity int) error {
	if quantity <= 0 {
		quantity = 1
	}

	// Verify item is currently in the wishlist
	exists, err := u.wishlistRepo.Exists(ctx, userID, bookID)
	if err != nil {
		return fmt.Errorf("failed checking wishlist: %w", err)
	}
	if !exists {
		return entity.ErrWishlistItemNotFound
	}

	// Verify book is valid and active
	book, err := u.bookRepo.FindByID(ctx, bookID)
	if err != nil {
		return fmt.Errorf("failed fetching book: %w", err)
	}
	if book == nil || !book.IsActive {
		return entity.ErrBookNotActiveInWishlist
	}
	if quantity > book.Stock {
		return entity.ErrInsufficientStock
	}

	// Add to cart
	cartInput := AddToCartInput{
		BookID:   bookID,
		Quantity: quantity,
	}
	if err := u.cartUsecase.AddToCart(ctx, userID, cartInput); err != nil {
		return err
	}

	// Remove from wishlist
	if err := u.wishlistRepo.DeleteByUserAndBook(ctx, userID, bookID); err != nil {
		return fmt.Errorf("failed removing from wishlist after move to cart: %w", err)
	}

	return nil
}
