package dto

type CreateAuthorRequest struct {
	Name string  `json:"name" validate:"required,min=2,max=150"`
	Bio  *string `json:"bio" validate:"omitempty,max=2000"`
}

type UpdateAuthorRequest struct {
	Name string  `json:"name" validate:"required,min=2,max=150"`
	Bio  *string `json:"bio" validate:"omitempty,max=2000"`
}
