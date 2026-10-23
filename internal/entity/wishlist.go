package entity

import (
	"errors"
	"time"
)

var (
	ErrWishlistItemNotFound    = errors.New("wishlist item not found")
	ErrWishlistAlreadyExists   = errors.New("book is already in wishlist")
	ErrBookNotActiveInWishlist = errors.New("book is no longer active or available")
)

type Wishlist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:uq_wishlists_user_book" json:"user_id"`
	BookID    uint      `gorm:"not null;uniqueIndex:uq_wishlists_user_book" json:"book_id"`
	Book      Book      `gorm:"foreignKey:BookID" json:"book"`
	CreatedAt time.Time `json:"created_at"`
}

func (Wishlist) TableName() string {
	return "wishlists"
}
