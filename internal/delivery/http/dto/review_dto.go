package dto

import (
	"strings"
	"time"
)

type BookReviewItemResponse struct {
	ID            uint       `json:"id"`
	Rating        int        `json:"rating"`
	Comment       *string    `json:"comment,omitempty"`
	UserName      string     `json:"user_name"`
	SellerReply   *string    `json:"seller_reply,omitempty"`
	SellerReplyAt *time.Time `json:"seller_reply_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type BookReviewBreakdownResponse struct {
	AverageRating    float64 `json:"average_rating"`
	TotalReviews     int64   `json:"total_reviews"`
	Star5Count       int64   `json:"star_5_count"`
	Star4Count       int64   `json:"star_4_count"`
	Star3Count       int64   `json:"star_3_count"`
	Star2Count       int64   `json:"star_2_count"`
	Star1Count       int64   `json:"star_1_count"`
	WithCommentCount int64   `json:"with_comment_count"`
}

type BookReviewsSummaryResponse struct {
	Breakdown BookReviewBreakdownResponse `json:"breakdown"`
	Reviews   []BookReviewItemResponse    `json:"reviews"`
	Meta      interface{}                 `json:"meta"`
}

type OrderReviewItemEligibility struct {
	BookID     uint       `json:"book_id"`
	Title      string     `json:"title"`
	CoverImage *string    `json:"cover_image,omitempty"`
	Price      int        `json:"price"`
	IsReviewed bool       `json:"is_reviewed"`
	ReviewID   *uint      `json:"review_id,omitempty"`
	Rating     *int       `json:"rating,omitempty"`
	Comment    *string    `json:"comment,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}

type OrderReviewEligibilityResponse struct {
	OrderID        uint                         `json:"order_id"`
	InvoiceNumber  string                       `json:"invoice_number"`
	OrderStatus    string                       `json:"order_status"`
	IsEligible     bool                         `json:"is_eligible"`
	DeliveredAt    *time.Time                   `json:"delivered_at,omitempty"`
	ReviewDeadline *time.Time                   `json:"review_deadline,omitempty"`
	RemainingDays  int                          `json:"remaining_days"`
	RemainingHours int                          `json:"remaining_hours"`
	IsExpired      bool                         `json:"is_expired"`
	Items          []OrderReviewItemEligibility `json:"items"`
}

type CreateReviewItemRequest struct {
	BookID  uint   `json:"book_id" validate:"required,gt=0"`
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"max=1000"`
}

type SubmitOrderReviewsRequest struct {
	Reviews []CreateReviewItemRequest `json:"reviews" validate:"required,min=1,dive"`
}

type UpdateReviewRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"max=1000"`
}

type CustomerReviewItemResponse struct {
	ID            uint       `json:"id"`
	BookID        uint       `json:"book_id"`
	BookTitle     string     `json:"book_title"`
	BookSlug      string     `json:"book_slug"`
	BookCover     *string    `json:"book_cover,omitempty"`
	OrderID       uint       `json:"order_id"`
	InvoiceNumber string     `json:"invoice_number"`
	Rating        int        `json:"rating"`
	Comment       *string    `json:"comment,omitempty"`
	SellerReply   *string    `json:"seller_reply,omitempty"`
	SellerReplyAt *time.Time `json:"seller_reply_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CanEdit       bool       `json:"can_edit"`
}

type AdminReviewItemResponse struct {
	ID            uint       `json:"id"`
	BookID        uint       `json:"book_id"`
	BookTitle     string     `json:"book_title"`
	BookCover     *string    `json:"book_cover,omitempty"`
	OrderID       uint       `json:"order_id"`
	InvoiceNumber string     `json:"invoice_number"`
	UserID        uint       `json:"user_id"`
	UserName      string     `json:"user_name"`
	UserEmail     string     `json:"user_email"`
	Rating        int        `json:"rating"`
	Comment       *string    `json:"comment,omitempty"`
	IsHidden      bool       `json:"is_hidden"`
	HiddenReason  *string    `json:"hidden_reason,omitempty"`
	SellerReply   *string    `json:"seller_reply,omitempty"`
	SellerReplyAt *time.Time `json:"seller_reply_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AdminModerateReviewRequest struct {
	IsHidden bool   `json:"is_hidden"`
	Reason   string `json:"reason" validate:"max=255"`
}

type AdminReplyReviewRequest struct {
	Reply string `json:"reply" validate:"required,min=1,max=1000"`
}

// MaskUserName masks part of the user name for public reviews with ****:
// e.g. "Budi" -> "B****i", "Alice" -> "A****e", "Budi Santoso" -> "B****i S****o"
func MaskUserName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Pengguna"
	}

	words := strings.Fields(name)
	if len(words) == 0 {
		return "Pengguna"
	}

	maskedWords := make([]string, len(words))
	for i, w := range words {
		runes := []rune(w)
		n := len(runes)
		if n <= 1 {
			maskedWords[i] = string(runes[0]) + "****"
		} else if n == 2 {
			maskedWords[i] = string(runes[0]) + "****" + string(runes[1])
		} else {
			maskedWords[i] = string(runes[0]) + "****" + string(runes[n-1])
		}
	}
	return strings.Join(maskedWords, " ")
}
