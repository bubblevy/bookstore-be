package dto

import "time"

type DiscountTierRequest struct {
	MinSpend    int     `json:"min_spend" validate:"required,gte=0"`
	RewardType  string  `json:"reward_type" validate:"required,oneof=PERCENTAGE FIXED"`
	RewardValue int     `json:"reward_value" validate:"required,gt=0"`
	MaxReward   *int    `json:"max_reward" validate:"omitempty,gt=0"`
}

type CreateDiscountRequest struct {
	Code              string                `json:"code" validate:"required,min=3,max=50,alphanum"`
	Name              string                `json:"name" validate:"required,min=3,max=150"`
	Description       *string               `json:"description" validate:"omitempty"`
	DiscountType      string                `json:"discount_type" validate:"required,oneof=PERCENTAGE FIXED TIERED"`
	DiscountValue     int                   `json:"discount_value" validate:"required,gt=0"`
	MinPurchase       int                   `json:"min_purchase" validate:"gte=0"`
	MaxDiscountAmount *int                  `json:"max_discount_amount" validate:"omitempty,gt=0"`
	Quota             int                   `json:"quota" validate:"gte=0"`
	IsAllBooks        *bool                 `json:"is_all_books" validate:"required"`
	IsActive          *bool                 `json:"is_active"`
	StartDate         time.Time             `json:"start_date" validate:"required"`
	EndDate           time.Time             `json:"end_date" validate:"required,gtfield=StartDate"`
	BookIDs           []uint                `json:"book_ids" validate:"omitempty"`
	Tiers             []DiscountTierRequest `json:"tiers" validate:"omitempty,dive"`
}

type UpdateDiscountRequest struct {
	Code              string                `json:"code" validate:"required,min=3,max=50,alphanum"`
	Name              string                `json:"name" validate:"required,min=3,max=150"`
	Description       *string               `json:"description" validate:"omitempty"`
	DiscountType      string                `json:"discount_type" validate:"required,oneof=PERCENTAGE FIXED TIERED"`
	DiscountValue     int                   `json:"discount_value" validate:"required,gt=0"`
	MinPurchase       int                   `json:"min_purchase" validate:"gte=0"`
	MaxDiscountAmount *int                  `json:"max_discount_amount" validate:"omitempty,gt=0"`
	Quota             int                   `json:"quota" validate:"gte=0"`
	IsAllBooks        *bool                 `json:"is_all_books" validate:"required"`
	IsActive          *bool                 `json:"is_active"`
	StartDate         time.Time             `json:"start_date" validate:"required"`
	EndDate           time.Time             `json:"end_date" validate:"required,gtfield=StartDate"`
	BookIDs           []uint                `json:"book_ids" validate:"omitempty"`
	Tiers             []DiscountTierRequest `json:"tiers" validate:"omitempty,dive"`
}

type CheckDiscountRequest struct {
	Code  string             `json:"code" validate:"required"`
	Items []CartItemForCheck `json:"items" validate:"required,min=1,dive"`
}

type CartItemForCheck struct {
	BookID   uint `json:"book_id" validate:"required"`
	Quantity int  `json:"quantity" validate:"required,gt=0"`
}
