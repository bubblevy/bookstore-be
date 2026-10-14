package entity

import "gorm.io/gorm"

type OrderItem struct {
	gorm.Model
	OrderID     uint    `gorm:"not null;index" json:"order_id"`
	BookID      uint    `gorm:"not null" json:"book_id"`
	Quantity    int     `gorm:"not null" json:"quantity"`
	Price       float64 `gorm:"type:numeric(12,2);not null" json:"price"` // Snapshot price at the time of checkout
	WeightGrams int     `gorm:"not null" json:"weight_grams"`             // Snapshot weight at the time of checkout

	// Relasi (Belongs To)
	Book *Book `gorm:"foreignKey:BookID" json:"book,omitempty"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
