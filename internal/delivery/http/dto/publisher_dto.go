package dto

type CreatePublisherRequest struct {
	Name string `json:"name" validate:"required,min=2,max=150"`
}

type UpdatePublisherRequest struct {
	Name string `json:"name" validate:"required,min=2,max=150"`
}
