package dto

type CreateBookRequest struct {
	CategoryID      uint    `json:"category_id" validate:"required"`
	AuthorID        uint    `json:"author_id" validate:"required"`
	PublisherID     uint    `json:"publisher_id" validate:"required"`
	Title           string  `json:"title" validate:"required,min=2,max=255"`
	ISBN            *string `json:"isbn" validate:"omitempty,max=30"`
	Synopsis        *string `json:"synopsis" validate:"omitempty"`
	Price           float64 `json:"price" validate:"required,gte=0"`
	WeightGrams     int     `json:"weight_grams" validate:"required,gt=0"`
	Pages           *int    `json:"pages" validate:"omitempty,gt=0"`
	PublicationYear *int    `json:"publication_year" validate:"omitempty,gte=1000,lte=2100"`
	Stock           int     `json:"stock" validate:"gte=0"`
	CoverImage      *string `json:"cover_image" validate:"omitempty,max=255"`
	IsActive        *bool   `json:"is_active"`
}

type UpdateBookRequest struct {
	CategoryID      uint    `json:"category_id" validate:"required"`
	AuthorID        uint    `json:"author_id" validate:"required"`
	PublisherID     uint    `json:"publisher_id" validate:"required"`
	Title           string  `json:"title" validate:"required,min=2,max=255"`
	ISBN            *string `json:"isbn" validate:"omitempty,max=30"`
	Synopsis        *string `json:"synopsis" validate:"omitempty"`
	Price           float64 `json:"price" validate:"required,gte=0"`
	WeightGrams     int     `json:"weight_grams" validate:"required,gt=0"`
	Pages           *int    `json:"pages" validate:"omitempty,gt=0"`
	PublicationYear *int    `json:"publication_year" validate:"omitempty,gte=1000,lte=2100"`
	Stock           int     `json:"stock" validate:"gte=0"`
	CoverImage      *string `json:"cover_image" validate:"omitempty,max=255"`
	IsActive        *bool   `json:"is_active"`
}
