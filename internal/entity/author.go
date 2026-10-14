package entity

import "gorm.io/gorm"

type Author struct {
	gorm.Model
	Name  string  `gorm:"type:varchar(150);not null" json:"name"`
	Bio   *string `gorm:"type:text" json:"bio,omitempty"`
	Books []Book  `gorm:"foreignKey:AuthorID" json:"books,omitempty"`
}

func (Author) TableName() string {
	return "authors"
}
