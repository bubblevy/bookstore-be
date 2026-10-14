package entity

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrPublisherNotFound     = errors.New("publisher not found")
	ErrPublisherSlugConflict = errors.New("publisher slug already exists")
)

type Publisher struct {
	gorm.Model
	Name  string `gorm:"type:varchar(150);not null" json:"name"`
	Slug  string `gorm:"type:varchar(170);not null;uniqueIndex:uq_publishers_slug_active,where:deleted_at IS NULL" json:"slug"`
	Books []Book `gorm:"foreignKey:PublisherID" json:"books,omitempty"`
}

func (Publisher) TableName() string {
	return "publishers"
}
