package entity

import (
	"errors"
	"time"
)

var (
	ErrCartItemNotFound    = errors.New("cart item not found")
	ErrCartEmpty           = errors.New("cart is empty")
	ErrInsufficientStock   = errors.New("book stock is insufficient")
	ErrBookNotActiveInCart = errors.New("book is no longer active or available")
)

type Cart struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;uniqueIndex" json:"user_id"`
	Items     []CartItem `gorm:"foreignKey:CartID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CartID    uint      `gorm:"not null;uniqueIndex:uq_cart_items_cart_book" json:"cart_id"`
	BookID    uint      `gorm:"not null;uniqueIndex:uq_cart_items_cart_book" json:"book_id"`
	Quantity  int       `gorm:"not null;default:1" json:"quantity"`
	Book      Book      `gorm:"foreignKey:BookID" json:"book"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
