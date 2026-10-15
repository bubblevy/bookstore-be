package dto

type AddToCartRequest struct {
	BookID   uint `json:"book_id" validate:"required,gt=0"`
	Quantity int  `json:"quantity" validate:"required,gt=0"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

type CartBookSummary struct {
	ID          uint    `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Price       float64 `json:"price"`
	WeightGrams int     `json:"weight_grams"`
	Stock       int     `json:"stock"`
	CoverImage  *string `json:"cover_image,omitempty"`
	IsActive    bool    `json:"is_active"`
}

type CartItemResponse struct {
	ID        uint            `json:"id"`
	BookID    uint            `json:"book_id"`
	Quantity  int             `json:"quantity"`
	Subtotal  float64         `json:"subtotal"`
	Subweight int             `json:"subweight_grams"`
	Book      CartBookSummary `json:"book"`
}

type CartResponse struct {
	ID          uint               `json:"id"`
	TotalItems  int                `json:"total_items"`
	TotalQty    int                `json:"total_quantity"`
	TotalWeight int                `json:"total_weight_grams"`
	TotalPrice  float64            `json:"total_price"`
	Items       []CartItemResponse `json:"items"`
}

type SelectedCartItemsRequest struct {
	ItemIDs []uint `json:"item_ids" validate:"required,min=1"`
}
