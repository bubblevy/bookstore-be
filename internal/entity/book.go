package entity

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrBookNotFound     = errors.New("book not found")
	ErrBookSlugConflict = errors.New("book slug already exists")
	ErrBookISBNConflict = errors.New("book isbn already exists")
	ErrBookOutOfStock   = errors.New("book out of stock")
)

type Book struct {
	gorm.Model
	CategoryID      uint    `gorm:"not null;index" json:"category_id"`
	AuthorID        uint    `gorm:"not null;index" json:"author_id"`
	PublisherID     uint    `gorm:"not null" json:"publisher_id"`
	Title           string  `gorm:"type:varchar(255);not null" json:"title"`
	Slug            string  `gorm:"type:varchar(270);not null;uniqueIndex:uq_books_slug_active,where:deleted_at IS NULL" json:"slug"`
	ISBN            *string `gorm:"type:varchar(30);uniqueIndex:uq_books_isbn_active,where:deleted_at IS NULL AND isbn IS NOT NULL" json:"isbn,omitempty"`
	Synopsis        *string `gorm:"type:text" json:"synopsis,omitempty"`
	Price           float64 `gorm:"type:numeric(12,2);not null" json:"price"`
	WeightGrams     int     `gorm:"not null" json:"weight_grams"`
	Pages           *int    `json:"pages,omitempty"`
	PublicationYear *int16  `gorm:"type:smallint" json:"publication_year,omitempty"`
	Stock           int     `gorm:"default:0;not null" json:"stock"`
	CoverImage      *string `gorm:"type:varchar(255)" json:"cover_image,omitempty"`
	IsActive        bool    `gorm:"default:true;not null;index" json:"is_active"`

	// Relasi (Belongs To)
	Category  *Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Author    *Author    `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Publisher *Publisher `gorm:"foreignKey:PublisherID" json:"publisher,omitempty"`
}

func (Book) TableName() string {
	return "books"
}
