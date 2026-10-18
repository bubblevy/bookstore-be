package dto

import "time"

type CreateCategoryRequest struct {
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	ImageURL *string `json:"image_url" validate:"omitempty,max=255"`
}

type UpdateCategoryRequest struct {
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	ImageURL *string `json:"image_url" validate:"omitempty,max=255"`
}

type CategoryResponse struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	ImageURL   *string   `json:"image_url,omitempty"`
	TotalBooks int64     `json:"total_books"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
