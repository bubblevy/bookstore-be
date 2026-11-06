package entity

import (
	"errors"
	"time"
)

var (
	ErrReviewNotFound         = errors.New("review not found")
	ErrReviewNotEligible      = errors.New("order is not eligible for review (order must be completed)")
	ErrRatingPeriodExpired    = errors.New("review submission period has expired (reviews are only allowed within 7 days after delivery)")
	ErrReviewAlreadySubmitted = errors.New("review has already been submitted for this book in this order")
	ErrInvalidRatingScore     = errors.New("rating score must be between 1 and 5")
	ErrBookNotInOrder         = errors.New("book is not part of this order")
	ErrUnauthorizedReview     = errors.New("you are not authorized to modify this review")
)

type Review struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"not null;index" json:"user_id"`
	BookID        uint       `gorm:"not null;index" json:"book_id"`
	OrderID       uint       `gorm:"not null;index" json:"order_id"`
	Rating        int        `gorm:"not null" json:"rating"`
	Comment       *string    `gorm:"type:text" json:"comment,omitempty"`
	IsHidden      bool       `gorm:"not null;default:false;index" json:"is_hidden"`
	HiddenReason  *string    `gorm:"type:varchar(255)" json:"hidden_reason,omitempty"`
	SellerReply   *string    `gorm:"type:text" json:"seller_reply,omitempty"`
	SellerReplyAt *time.Time `json:"seller_reply_at,omitempty"`
	CreatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relations
	User  *User  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Book  *Book  `gorm:"foreignKey:BookID;constraint:OnDelete:CASCADE" json:"book,omitempty"`
	Order *Order `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"order,omitempty"`
}

func (Review) TableName() string {
	return "reviews"
}

