package entity

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrBookNotFound          = errors.New("book not found")
	ErrBookSlugConflict      = errors.New("book slug already exists")
	ErrBookISBNConflict      = errors.New("book isbn already exists")
	ErrBookInsufficientStock = errors.New("insufficient book stock")
	ErrBookCategoryNotFound  = errors.New("book category not found")
	ErrBookAuthorNotFound    = errors.New("book author not found")
	ErrBookPublisherNotFound = errors.New("book publisher not found")
)

type Book struct {
	gorm.Model
	CategoryID  uint `gorm:"not null;index" json:"category_id"`
	AuthorID    uint `gorm:"not null;index" json:"author_id"`
	PublisherID uint `gorm:"not null;index" json:"publisher_id"`

	Title           string  `gorm:"type:varchar(255);not null" json:"title"`
	Slug            string  `gorm:"type:varchar(270);not null;uniqueIndex:uq_books_slug_active,where:deleted_at IS NULL" json:"slug"`
	ISBN            *string `gorm:"type:varchar(30);uniqueIndex:uq_books_isbn_active,where:deleted_at IS NULL AND isbn IS NOT NULL" json:"isbn,omitempty"`
	Synopsis        *string `gorm:"type:text" json:"synopsis,omitempty"`
	Price           float64 `gorm:"type:numeric(12,2);not null" json:"price"`
	WeightGrams     int     `gorm:"not null" json:"weight_grams"`
	Pages           *int    `json:"pages,omitempty"`
	PublicationYear *int    `gorm:"type:smallint" json:"publication_year,omitempty"`
	Stock           int     `gorm:"not null;default:0" json:"stock"`
	CoverImage      *string `gorm:"type:varchar(255)" json:"cover_image,omitempty"`
	IsActive        bool    `gorm:"not null;default:true" json:"is_active"`

	// Belongs-To Relations
	Category  Category  `gorm:"foreignKey:CategoryID;constraint:OnDelete:RESTRICT;" json:"category,omitempty"`
	Author    Author    `gorm:"foreignKey:AuthorID;constraint:OnDelete:RESTRICT;" json:"author,omitempty"`
	Publisher Publisher `gorm:"foreignKey:PublisherID;constraint:OnDelete:RESTRICT;" json:"publisher,omitempty"`

	// Many-to-Many Pivot Table Discount
	Discounts []Discount `gorm:"many2many:discount_books;constraint:OnDelete:CASCADE;" json:"discounts,omitempty"`
}

func (Book) TableName() string {
	return "books"
}
