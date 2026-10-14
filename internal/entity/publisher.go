package entity

import "gorm.io/gorm"

type Publisher struct {
	gorm.Model
	Name  string `gorm:"type:varchar(150);not null" json:"name"`
	Books []Book `gorm:"foreignKey:PublisherID" json:"books,omitempty"`
}

func (Publisher) TableName() string {
	return "publishers"
}
