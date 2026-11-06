package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/pkg/pagination"
)

type ReviewFilter struct {
	BookID     *uint
	Rating     *int
	HasComment *bool
	IsHidden   *bool
	Search     string
	SortBy     string // "newest", "oldest", "highest_rating", "lowest_rating"
}

type ReviewStats struct {
	AverageRating    float64
	TotalReviews     int64
	Star5Count       int64
	Star4Count       int64
	Star3Count       int64
	Star2Count       int64
	Star1Count       int64
	WithCommentCount int64
}

type ReviewRepository interface {
	Create(ctx context.Context, review *entity.Review) error
	CreateBatch(ctx context.Context, reviews []entity.Review) error
	Update(ctx context.Context, review *entity.Review) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*entity.Review, error)
	FindByOrderAndBook(ctx context.Context, orderID, bookID uint) (*entity.Review, error)
	FindByOrderID(ctx context.Context, orderID uint) ([]entity.Review, error)
	FindBookReviews(ctx context.Context, bookID uint, filter ReviewFilter, pq pagination.Query) ([]entity.Review, int64, error)
	GetBookReviewStats(ctx context.Context, bookID uint) (*ReviewStats, error)
	FindUserReviews(ctx context.Context, userID uint, pq pagination.Query) ([]entity.Review, int64, error)
	FindAllAdmin(ctx context.Context, filter ReviewFilter, pq pagination.Query) ([]entity.Review, int64, error)
	RecalculateBookRating(ctx context.Context, bookID uint) error
}

