package dto

type CreateBookRequest struct {
	CategoryID         uint     `json:"category_id" validate:"required"`
	AuthorID           uint     `json:"author_id" validate:"required"`
	PublisherID        uint     `json:"publisher_id" validate:"required"`
	Title              string   `json:"title" validate:"required,min=2,max=255"`
	ISBN               string   `json:"isbn" validate:"required,max=30"`
	Synopsis           string   `json:"synopsis" validate:"required,min=5"`
	Price              int      `json:"price" validate:"required,gte=0"`
	OriginalPrice      *int     `json:"original_price" validate:"omitempty,gte=0"`
	DiscountPercentage *int     `json:"discount_percentage" validate:"omitempty,gte=0,lte=100"`
	WeightGrams        int      `json:"weight_grams" validate:"required,gt=0"`
	Pages              int      `json:"pages" validate:"required,gt=0"`
	PublicationYear    int      `json:"publication_year" validate:"required,gte=1000,lte=2100"`
	Language           string   `json:"language" validate:"required,min=2,max=50"`
	Length             float64  `json:"length" validate:"required,gt=0"`
	Width              float64  `json:"width" validate:"required,gt=0"`
	Height             float64  `json:"height" validate:"required,gt=0"`
	Condition          string   `json:"condition" validate:"required,min=2,max=100"`
	Stock              *int     `json:"stock" validate:"required,gte=0"`
	CoverImage         string   `json:"cover_image" validate:"required,max=255"`
	IsActive           *bool    `json:"is_active"`
}

type UpdateBookRequest struct {
	CategoryID         uint     `json:"category_id" validate:"required"`
	AuthorID           uint     `json:"author_id" validate:"required"`
	PublisherID        uint     `json:"publisher_id" validate:"required"`
	Title              string   `json:"title" validate:"required,min=2,max=255"`
	ISBN               string   `json:"isbn" validate:"required,max=30"`
	Synopsis           string   `json:"synopsis" validate:"required,min=5"`
	Price              int      `json:"price" validate:"required,gte=0"`
	OriginalPrice      *int     `json:"original_price" validate:"omitempty,gte=0"`
	DiscountPercentage *int     `json:"discount_percentage" validate:"omitempty,gte=0,lte=100"`
	WeightGrams        int      `json:"weight_grams" validate:"required,gt=0"`
	Pages              int      `json:"pages" validate:"required,gt=0"`
	PublicationYear    int      `json:"publication_year" validate:"required,gte=1000,lte=2100"`
	Language           string   `json:"language" validate:"required,min=2,max=50"`
	Length             float64  `json:"length" validate:"required,gt=0"`
	Width              float64  `json:"width" validate:"required,gt=0"`
	Height             float64  `json:"height" validate:"required,gt=0"`
	Condition          string   `json:"condition" validate:"required,min=2,max=100"`
	Stock              *int     `json:"stock" validate:"required,gte=0"`
	CoverImage         string   `json:"cover_image" validate:"required,max=255"`
	IsActive           *bool    `json:"is_active"`
}
