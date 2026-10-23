package dto

import "time"

type AddWishlistRequest struct {
	BookID uint `json:"book_id" validate:"required,gt=0"`
}

type MoveToCartRequest struct {
	Quantity int `json:"quantity" validate:"omitempty,gt=0"`
}

type WishlistBookSummary struct {
	ID                 uint    `json:"id"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Price              int     `json:"price"`
	OriginalPrice      *int    `json:"original_price,omitempty"`
	DiscountPercentage *int    `json:"discount_percentage,omitempty"`
	WeightGrams        int     `json:"weight_grams"`
	Stock              int     `json:"stock"`
	CoverImage         *string `json:"cover_image,omitempty"`
	IsActive           bool    `json:"is_active"`
	CategoryName       string  `json:"category_name,omitempty"`
	AuthorName         string  `json:"author_name,omitempty"`
	PublisherName      string  `json:"publisher_name,omitempty"`
}

type WishlistItemResponse struct {
	ID        uint                `json:"id"`
	BookID    uint                `json:"book_id"`
	Book      WishlistBookSummary `json:"book"`
	CreatedAt time.Time           `json:"created_at"`
}

type WishlistCheckResponse struct {
	BookID       uint `json:"book_id"`
	IsWishlisted bool `json:"is_wishlisted"`
}
