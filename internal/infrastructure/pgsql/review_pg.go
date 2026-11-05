package pgsql

import (
	"context"
	"errors"
	"strings"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"gorm.io/gorm"
)

type reviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) repository.ReviewRepository {
	return &reviewRepository{db: db}
}

func (r *reviewRepository) Create(ctx context.Context, review *entity.Review) error {
	db := extractDB(ctx, r.db)
	return db.WithContext(ctx).Create(review).Error
}

func (r *reviewRepository) CreateBatch(ctx context.Context, reviews []entity.Review) error {
	if len(reviews) == 0 {
		return nil
	}
	db := extractDB(ctx, r.db)
	return db.WithContext(ctx).Create(&reviews).Error
}

func (r *reviewRepository) Update(ctx context.Context, review *entity.Review) error {
	db := extractDB(ctx, r.db)
	return db.WithContext(ctx).Save(review).Error
}

func (r *reviewRepository) Delete(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	return db.WithContext(ctx).Delete(&entity.Review{}, id).Error
}

func (r *reviewRepository) FindByID(ctx context.Context, id uint) (*entity.Review, error) {
	db := extractDB(ctx, r.db)
	var review entity.Review
	err := db.WithContext(ctx).
		Preload("User").
		Preload("Book").
		Preload("Order").
		Where("id = ?", id).
		First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) FindByOrderAndBook(ctx context.Context, orderID, bookID uint) (*entity.Review, error) {
	db := extractDB(ctx, r.db)
	var review entity.Review
	err := db.WithContext(ctx).
		Preload("User").
		Preload("Book").
		Where("order_id = ? AND book_id = ?", orderID, bookID).
		First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) FindByOrderID(ctx context.Context, orderID uint) ([]entity.Review, error) {
	db := extractDB(ctx, r.db)
	var reviews []entity.Review
	err := db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Find(&reviews).Error
	if err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewRepository) FindBookReviews(ctx context.Context, bookID uint, filter repository.ReviewFilter, pq pagination.Query) ([]entity.Review, int64, error) {
	db := extractDB(ctx, r.db)

	query := db.WithContext(ctx).Model(&entity.Review{}).
		Where("book_id = ? AND is_hidden = false", bookID)

	if filter.Rating != nil && *filter.Rating >= 1 && *filter.Rating <= 5 {
		query = query.Where("rating = ?", *filter.Rating)
	}

	if filter.HasComment != nil && *filter.HasComment {
		query = query.Where("comment IS NOT NULL AND TRIM(comment) != ''")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderClause := "created_at DESC"
	switch filter.SortBy {
	case "oldest":
		orderClause = "created_at ASC"
	case "highest_rating":
		orderClause = "rating DESC, created_at DESC"
	case "lowest_rating":
		orderClause = "rating ASC, created_at DESC"
	}

	var reviews []entity.Review
	err := query.
		Preload("User").
		Order(orderClause).
		Limit(pq.PerPage).
		Offset(pq.Offset()).
		Find(&reviews).Error
	if err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *reviewRepository) GetBookReviewStats(ctx context.Context, bookID uint) (*repository.ReviewStats, error) {
	db := extractDB(ctx, r.db)

	type statsRow struct {
		AvgRating    float64 `gorm:"column:avg_rating"`
		TotalCount   int64   `gorm:"column:total_count"`
		Star5        int64   `gorm:"column:star_5"`
		Star4        int64   `gorm:"column:star_4"`
		Star3        int64   `gorm:"column:star_3"`
		Star2        int64   `gorm:"column:star_2"`
		Star1        int64   `gorm:"column:star_1"`
		WithComment  int64   `gorm:"column:with_comment"`
	}

	var row statsRow
	query := `
		SELECT
			COALESCE(ROUND(AVG(rating)::numeric, 2), 0.00)::float8 AS avg_rating,
			COUNT(*)::bigint AS total_count,
			COUNT(CASE WHEN rating = 5 THEN 1 END)::bigint AS star_5,
			COUNT(CASE WHEN rating = 4 THEN 1 END)::bigint AS star_4,
			COUNT(CASE WHEN rating = 3 THEN 1 END)::bigint AS star_3,
			COUNT(CASE WHEN rating = 2 THEN 1 END)::bigint AS star_2,
			COUNT(CASE WHEN rating = 1 THEN 1 END)::bigint AS star_1,
			COUNT(CASE WHEN comment IS NOT NULL AND TRIM(comment) != '' THEN 1 END)::bigint AS with_comment
		FROM reviews
		WHERE book_id = ? AND is_hidden = false
	`
	if err := db.WithContext(ctx).Raw(query, bookID).Scan(&row).Error; err != nil {
		return nil, err
	}

	return &repository.ReviewStats{
		AverageRating:    row.AvgRating,
		TotalReviews:     row.TotalCount,
		Star5Count:       row.Star5,
		Star4Count:       row.Star4,
		Star3Count:       row.Star3,
		Star2Count:       row.Star2,
		Star1Count:       row.Star1,
		WithCommentCount: row.WithComment,
	}, nil
}

func (r *reviewRepository) FindUserReviews(ctx context.Context, userID uint, pq pagination.Query) ([]entity.Review, int64, error) {
	db := extractDB(ctx, r.db)

	query := db.WithContext(ctx).Model(&entity.Review{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reviews []entity.Review
	err := query.
		Preload("Book").
		Preload("Order").
		Order("created_at DESC").
		Limit(pq.PerPage).
		Offset(pq.Offset()).
		Find(&reviews).Error
	if err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *reviewRepository) FindAllAdmin(ctx context.Context, filter repository.ReviewFilter, pq pagination.Query) ([]entity.Review, int64, error) {
	db := extractDB(ctx, r.db)

	query := db.WithContext(ctx).Model(&entity.Review{}).
		Joins("LEFT JOIN users ON users.id = reviews.user_id").
		Joins("LEFT JOIN books ON books.id = reviews.book_id").
		Joins("LEFT JOIN orders ON orders.id = reviews.order_id")

	if filter.BookID != nil && *filter.BookID > 0 {
		query = query.Where("reviews.book_id = ?", *filter.BookID)
	}

	if filter.Rating != nil && *filter.Rating >= 1 && *filter.Rating <= 5 {
		query = query.Where("reviews.rating = ?", *filter.Rating)
	}

	if filter.IsHidden != nil {
		query = query.Where("reviews.is_hidden = ?", *filter.IsHidden)
	}

	if trimmedSearch := strings.TrimSpace(filter.Search); trimmedSearch != "" {
		s := "%" + trimmedSearch + "%"
		query = query.Where("reviews.comment ILIKE ? OR books.title ILIKE ? OR users.name ILIKE ? OR orders.invoice_number ILIKE ?", s, s, s, s)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderClause := "reviews.created_at DESC"
	switch filter.SortBy {
	case "oldest":
		orderClause = "reviews.created_at ASC"
	case "highest_rating":
		orderClause = "reviews.rating DESC, reviews.created_at DESC"
	case "lowest_rating":
		orderClause = "reviews.rating ASC, reviews.created_at DESC"
	}

	var reviews []entity.Review
	err := query.
		Preload("User").
		Preload("Book").
		Preload("Order").
		Order(orderClause).
		Limit(pq.PerPage).
		Offset(pq.Offset()).
		Find(&reviews).Error
	if err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *reviewRepository) RecalculateBookRating(ctx context.Context, bookID uint) error {
	db := extractDB(ctx, r.db)

	type ratingResult struct {
		AvgRating  float64 `gorm:"column:avg_rating"`
		TotalCount int     `gorm:"column:total_count"`
	}

	var res ratingResult
	query := `
		SELECT
			COALESCE(ROUND(AVG(rating)::numeric, 2), 0.00)::float8 AS avg_rating,
			COUNT(*)::int AS total_count
		FROM reviews
		WHERE book_id = ? AND is_hidden = false
	`
	if err := db.WithContext(ctx).Raw(query, bookID).Scan(&res).Error; err != nil {
		return err
	}

	return db.WithContext(ctx).Model(&entity.Book{}).
		Where("id = ?", bookID).
		Updates(map[string]interface{}{
			"average_rating": res.AvgRating,
			"total_reviews":  res.TotalCount,
		}).Error
}

