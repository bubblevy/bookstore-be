package dto

import (
	"time"
)

type PaymentInstructionDTO struct {
	Title   string `json:"title" validate:"required"`
	Content string `json:"content" validate:"required"`
}

type PaymentMethodDetailResponse struct {
	ID           uint                    `json:"id"`
	Code         string                  `json:"code"`
	Name         string                  `json:"name"`
	Category     string                  `json:"category"`
	FeeType      string                  `json:"fee_type"`
	FeeAmount    int                     `json:"fee_amount"`
	FeePercent   float64                 `json:"fee_percent"`
	EstimatedFee int                     `json:"estimated_fee,omitempty"`
	Image        string                  `json:"image"`
	Description  string                  `json:"description"`
	Instructions []PaymentInstructionDTO `json:"instructions"`
	IsActive     bool                    `json:"is_active"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

type UpdatePaymentMethodRequest struct {
	Name         *string                 `json:"name,omitempty" form:"name" validate:"omitempty,min=1,max=100"`
	Category     *string                 `json:"category,omitempty" form:"category" validate:"omitempty,min=1,max=50"`
	FeeType      *string                 `json:"fee_type,omitempty" form:"fee_type" validate:"omitempty,oneof=fixed percentage"`
	FeeAmount    *int                    `json:"fee_amount,omitempty" form:"fee_amount" validate:"omitempty,gte=0"`
	FeePercent   *float64                `json:"fee_percent,omitempty" form:"fee_percent" validate:"omitempty,gte=0"`
	Image        *string                 `json:"image,omitempty" form:"image"`
	Description  *string                 `json:"description,omitempty" form:"description"`
	Instructions []PaymentInstructionDTO `json:"instructions,omitempty" form:"instructions"`
	IsActive     *bool                   `json:"is_active,omitempty" form:"is_active"`
}
