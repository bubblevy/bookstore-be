package entity

import "gorm.io/gorm"

type Category struct {
	gorm.Model
	Name  string `gorm:"type:varchar(100);not null" json:"name"`
	Slug  string `gorm:"type:varchar(120);uniqueIndex;not null" json:"slug"`
	Books []Book `gorm:"foreignKey:CategoryID" json:"books,omitempty"`
}

func (Category) TableName() string {
	return "categories"
}
