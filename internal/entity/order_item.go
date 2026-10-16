package entity

import "time"

type OrderItem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OrderID     uint      `gorm:"not null" json:"order_id"`
	BookID      uint      `gorm:"not null" json:"book_id"`
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`
	CoverImage  *string   `gorm:"type:varchar(255)" json:"cover_image,omitempty"`
	Price       float64   `gorm:"type:numeric(15,2);not null" json:"price"`
	WeightGrams int       `gorm:"not null" json:"weight_grams"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	Subtotal    float64   `gorm:"type:numeric(15,2);not null" json:"subtotal"`
	CreatedAt   time.Time `json:"created_at"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
