package entity

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategorySlugConflict = errors.New("category slug already exists")
)

type Category struct {
	gorm.Model
	Name  string `gorm:"type:varchar(100);not null" json:"name"`
	Slug  string `gorm:"type:varchar(120);not null;uniqueIndex:uq_categories_slug_active,where:deleted_at IS NULL" json:"slug"`
	Books []Book `gorm:"foreignKey:CategoryID" json:"books,omitempty"`
}

func (Category) TableName() string {
	return "categories"
}
