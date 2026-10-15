package entity

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrAuthorNotFound     = errors.New("author not found")
	ErrAuthorSlugConflict = errors.New("author slug already exists")
)

type Author struct {
	gorm.Model
	Name  string  `gorm:"type:varchar(150);not null" json:"name"`
	Slug  string  `gorm:"type:varchar(170);not null;uniqueIndex:uq_authors_slug_active,where:deleted_at IS NULL" json:"slug"`
	Bio   *string `gorm:"type:text" json:"bio,omitempty"`
	Books []Book  `gorm:"foreignKey:AuthorID" json:"books,omitempty"`
}

func (Author) TableName() string {
	return "authors"
}
