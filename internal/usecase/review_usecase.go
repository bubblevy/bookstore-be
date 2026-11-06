package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/cache"
	"github.com/bubblevy/bookstore/pkg/pagination"
)

const ReviewEligibilityWindow = 7 * 24 * time.Hour

type ReviewUsecase interface {
	GetBookReviews(ctx context.Context, slug string, filter repository.ReviewFilter, pq pagination.Query) (*dto.BookReviewsSummaryResponse, error)
	CheckOrderReviewEligibility(ctx context.Context, userID uint, invoiceNumber string) (*dto.OrderReviewEligibilityResponse, error)
	SubmitOrderReviews(ctx context.Context, userID uint, invoiceNumber string, req dto.SubmitOrderReviewsRequest) ([]dto.CustomerReviewItemResponse, error)
	UpdateReview(ctx context.Context, userID uint, reviewID uint, req dto.UpdateReviewRequest) (*dto.CustomerReviewItemResponse, error)
	GetCustomerReviews(ctx context.Context, userID uint, pq pagination.Query) ([]dto.CustomerReviewItemResponse, pagination.Meta, error)
	AdminFindAllReviews(ctx context.Context, filter repository.ReviewFilter, pq pagination.Query) ([]dto.AdminReviewItemResponse, pagination.Meta, error)
	AdminModerateReview(ctx context.Context, reviewID uint, req dto.AdminModerateReviewRequest) error
	AdminReplyReview(ctx context.Context, reviewID uint, req dto.AdminReplyReviewRequest) error
	AdminDeleteReview(ctx context.Context, reviewID uint) error
}

type reviewUsecase struct {
	reviewRepo repository.ReviewRepository
	orderRepo  repository.OrderRepository
	bookRepo   repository.BookRepository
	txManager  repository.TransactionManager
	cacheSvc   cache.Cache
}

func NewReviewUsecase(
	reviewRepo repository.ReviewRepository,
	orderRepo repository.OrderRepository,
	bookRepo repository.BookRepository,
	txManager repository.TransactionManager,
	cacheSvc cache.Cache,
) ReviewUsecase {
	return &reviewUsecase{
		reviewRepo: reviewRepo,
		orderRepo:  orderRepo,
		bookRepo:   bookRepo,
		txManager:  txManager,
		cacheSvc:   cacheSvc,
	}
}

func (u *reviewUsecase) invalidateBookCache(ctx context.Context) {
	if u.cacheSvc != nil {
		_ = u.cacheSvc.DeletePrefix(ctx, "books:")
	}
}

func (u *reviewUsecase) GetBookReviews(ctx context.Context, slug string, filter repository.ReviewFilter, pq pagination.Query) (*dto.BookReviewsSummaryResponse, error) {
	cleanSlug := strings.TrimSpace(slug)
	if cleanSlug == "" {
		return nil, entity.ErrBookNotFound
	}

	book, err := u.bookRepo.FindBySlug(ctx, cleanSlug)
	if err != nil {
		return nil, entity.ErrBookNotFound
	}

	stats, err := u.reviewRepo.GetBookReviewStats(ctx, book.ID)
	if err != nil {
		return nil, fmt.Errorf("failed fetching review statistics: %w", err)
	}

	reviews, total, err := u.reviewRepo.FindBookReviews(ctx, book.ID, filter, pq)
	if err != nil {
		return nil, fmt.Errorf("failed fetching book reviews: %w", err)
	}

	items := make([]dto.BookReviewItemResponse, len(reviews))
	for i, r := range reviews {
		userName := "Pengguna"
		if r.User != nil {
			userName = dto.MaskUserName(r.User.Name)
		}

		items[i] = dto.BookReviewItemResponse{
			ID:            r.ID,
			Rating:        r.Rating,
			Comment:       r.Comment,
			UserName:      userName,
			SellerReply:   r.SellerReply,
			SellerReplyAt: r.SellerReplyAt,
			CreatedAt:     r.CreatedAt,
		}
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)

	return &dto.BookReviewsSummaryResponse{
		Breakdown: dto.BookReviewBreakdownResponse{
			AverageRating:    stats.AverageRating,
			TotalReviews:     stats.TotalReviews,
			Star5Count:       stats.Star5Count,
			Star4Count:       stats.Star4Count,
			Star3Count:       stats.Star3Count,
			Star2Count:       stats.Star2Count,
			Star1Count:       stats.Star1Count,
			WithCommentCount: stats.WithCommentCount,
		},
		Reviews: items,
		Meta:    meta,
	}, nil
}

func (u *reviewUsecase) CheckOrderReviewEligibility(ctx context.Context, userID uint, invoiceNumber string) (*dto.OrderReviewEligibilityResponse, error) {
	order, err := u.orderRepo.FindOrderDetailByInvoice(ctx, userID, invoiceNumber)
	if err != nil {
		return nil, entity.ErrOrderNotFound
	}

	var deliveredAt *time.Time
	if order.CompletedAt != nil {
		deliveredAt = order.CompletedAt
	} else if order.Status == entity.OrderStatusCompleted {
		deliveredAt = &order.UpdatedAt
	}

	var deadline *time.Time
	isExpired := false
	remainingDays := 0
	remainingHours := 0
	isEligible := false

	now := time.Now()

	if deliveredAt != nil {
		dl := deliveredAt.Add(ReviewEligibilityWindow)
		deadline = &dl

		if now.After(dl) {
			isExpired = true
		} else {
			durationLeft := dl.Sub(now)
			remainingDays = int(durationLeft.Hours() / 24)
			remainingHours = int(durationLeft.Hours())
		}
	}

	if order.Status == entity.OrderStatusCompleted && !isExpired && deliveredAt != nil {
		isEligible = true
	}

	existingReviews, err := u.reviewRepo.FindByOrderID(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed fetching existing reviews: %w", err)
	}

	reviewMap := make(map[uint]*entity.Review)
	for i := range existingReviews {
		reviewMap[existingReviews[i].BookID] = &existingReviews[i]
	}

	items := make([]dto.OrderReviewItemEligibility, len(order.Items))
	for i, it := range order.Items {
		itemElig := dto.OrderReviewItemEligibility{
			BookID:     it.BookID,
			Title:      it.Title,
			CoverImage: it.CoverImage,
			Price:      it.Price,
			IsReviewed: false,
		}

		if rev, exists := reviewMap[it.BookID]; exists {
			itemElig.IsReviewed = true
			itemElig.ReviewID = &rev.ID
			itemElig.Rating = &rev.Rating
			itemElig.Comment = rev.Comment
			itemElig.CreatedAt = &rev.CreatedAt
		}

		items[i] = itemElig
	}

	return &dto.OrderReviewEligibilityResponse{
		OrderID:        order.ID,
		InvoiceNumber:  order.InvoiceNumber,
		OrderStatus:    string(order.Status),
		IsEligible:     isEligible,
		DeliveredAt:    deliveredAt,
		ReviewDeadline: deadline,
		RemainingDays:  remainingDays,
		RemainingHours: remainingHours,
		IsExpired:      isExpired,
		Items:          items,
	}, nil
}

func (u *reviewUsecase) SubmitOrderReviews(ctx context.Context, userID uint, invoiceNumber string, req dto.SubmitOrderReviewsRequest) ([]dto.CustomerReviewItemResponse, error) {
	if len(req.Reviews) == 0 {
		return nil, errors.New("reviews list cannot be empty")
	}

	order, err := u.orderRepo.FindOrderDetailByInvoice(ctx, userID, invoiceNumber)
	if err != nil {
		return nil, entity.ErrOrderNotFound
	}

	if order.Status != entity.OrderStatusCompleted {
		return nil, entity.ErrReviewNotEligible
	}

	var deliveredAt time.Time
	if order.CompletedAt != nil {
		deliveredAt = *order.CompletedAt
	} else {
		deliveredAt = order.UpdatedAt
	}

	if time.Now().After(deliveredAt.Add(ReviewEligibilityWindow)) {
		return nil, entity.ErrRatingPeriodExpired
	}

	orderItemMap := make(map[uint]entity.OrderItem)
	for _, it := range order.Items {
		orderItemMap[it.BookID] = it
	}

	// Validate all submitted reviews against order items and validate rating scores
	seenBooks := make(map[uint]bool)
	for _, item := range req.Reviews {
		if item.Rating < 1 || item.Rating > 5 {
			return nil, entity.ErrInvalidRatingScore
		}
		if _, exists := orderItemMap[item.BookID]; !exists {
			return nil, fmt.Errorf("%w: book id %d", entity.ErrBookNotInOrder, item.BookID)
		}
		if seenBooks[item.BookID] {
			return nil, fmt.Errorf("duplicate book id %d in review submission request", item.BookID)
		}
		seenBooks[item.BookID] = true

		// Check if already reviewed
		existing, err := u.reviewRepo.FindByOrderAndBook(ctx, order.ID, item.BookID)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("%w for book %d", entity.ErrReviewAlreadySubmitted, item.BookID)
		}
	}

	var createdReviews []entity.Review
	now := time.Now()

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, item := range req.Reviews {
			var commentPtr *string
			if trimmed := strings.TrimSpace(item.Comment); trimmed != "" {
				commentPtr = &trimmed
			}

			review := entity.Review{
				UserID:      userID,
				BookID:      item.BookID,
				OrderID:     order.ID,
				Rating:      item.Rating,
				Comment:     commentPtr,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			if err := u.reviewRepo.Create(txCtx, &review); err != nil {
				return fmt.Errorf("failed creating review for book %d: %w", item.BookID, err)
			}

			if err := u.reviewRepo.RecalculateBookRating(txCtx, item.BookID); err != nil {
				return fmt.Errorf("failed recalculating rating for book %d: %w", item.BookID, err)
			}

			createdReviews = append(createdReviews, review)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	u.invalidateBookCache(ctx)

	results := make([]dto.CustomerReviewItemResponse, len(createdReviews))
	for i, r := range createdReviews {
		bookTitle := ""
		bookCover := (*string)(nil)
		if item, ok := orderItemMap[r.BookID]; ok {
			bookTitle = item.Title
			bookCover = item.CoverImage
		}

		results[i] = dto.CustomerReviewItemResponse{
			ID:            r.ID,
			BookID:        r.BookID,
			BookTitle:     bookTitle,
			BookCover:     bookCover,
			OrderID:       order.ID,
			InvoiceNumber: order.InvoiceNumber,
			Rating:        r.Rating,
			Comment:       r.Comment,
			CreatedAt:     r.CreatedAt,
			CanEdit:       true,
		}
	}

	return results, nil
}

func (u *reviewUsecase) UpdateReview(ctx context.Context, userID uint, reviewID uint, req dto.UpdateReviewRequest) (*dto.CustomerReviewItemResponse, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, entity.ErrInvalidRatingScore
	}

	review, err := u.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}

	if review.UserID != userID {
		return nil, entity.ErrUnauthorizedReview
	}

	if review.Order == nil {
		ord, err := u.orderRepo.FindByID(ctx, review.OrderID)
		if err != nil {
			return nil, entity.ErrOrderNotFound
		}
		review.Order = ord
	}

	var deliveredAt time.Time
	if review.Order.CompletedAt != nil {
		deliveredAt = *review.Order.CompletedAt
	} else {
		deliveredAt = review.Order.UpdatedAt
	}

	if time.Now().After(deliveredAt.Add(ReviewEligibilityWindow)) {
		return nil, entity.ErrRatingPeriodExpired
	}

	var commentPtr *string
	if trimmed := strings.TrimSpace(req.Comment); trimmed != "" {
		commentPtr = &trimmed
	}

	review.Rating = req.Rating
	review.Comment = commentPtr
	review.UpdatedAt = time.Now()

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := u.reviewRepo.Update(txCtx, review); err != nil {
			return err
		}
		return u.reviewRepo.RecalculateBookRating(txCtx, review.BookID)
	})

	if err != nil {
		return nil, err
	}

	u.invalidateBookCache(ctx)

	bookTitle := ""
	bookSlug := ""
	var bookCover *string
	if review.Book != nil {
		bookTitle = review.Book.Title
		bookSlug = review.Book.Slug
		bookCover = review.Book.CoverImage
	}

	invoiceNum := ""
	if review.Order != nil {
		invoiceNum = review.Order.InvoiceNumber
	}

	return &dto.CustomerReviewItemResponse{
		ID:            review.ID,
		BookID:        review.BookID,
		BookTitle:     bookTitle,
		BookSlug:      bookSlug,
		BookCover:     bookCover,
		OrderID:       review.OrderID,
		InvoiceNumber: invoiceNum,
		Rating:        review.Rating,
		Comment:       review.Comment,
		SellerReply:   review.SellerReply,
		SellerReplyAt: review.SellerReplyAt,
		CreatedAt:     review.CreatedAt,
		CanEdit:       true,
	}, nil
}

func (u *reviewUsecase) GetCustomerReviews(ctx context.Context, userID uint, pq pagination.Query) ([]dto.CustomerReviewItemResponse, pagination.Meta, error) {
	reviews, total, err := u.reviewRepo.FindUserReviews(ctx, userID, pq)
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed fetching customer reviews: %w", err)
	}

	now := time.Now()
	items := make([]dto.CustomerReviewItemResponse, len(reviews))
	for i, r := range reviews {
		bookTitle := ""
		bookSlug := ""
		var bookCover *string
		if r.Book != nil {
			bookTitle = r.Book.Title
			bookSlug = r.Book.Slug
			bookCover = r.Book.CoverImage
		}

		invoiceNum := ""
		canEdit := false
		if r.Order != nil {
			invoiceNum = r.Order.InvoiceNumber
			var deliveredAt time.Time
			if r.Order.CompletedAt != nil {
				deliveredAt = *r.Order.CompletedAt
			} else {
				deliveredAt = r.Order.UpdatedAt
			}
			canEdit = now.Before(deliveredAt.Add(ReviewEligibilityWindow))
		}

		items[i] = dto.CustomerReviewItemResponse{
			ID:            r.ID,
			BookID:        r.BookID,
			BookTitle:     bookTitle,
			BookSlug:      bookSlug,
			BookCover:     bookCover,
			OrderID:       r.OrderID,
			InvoiceNumber: invoiceNum,
			Rating:        r.Rating,
			Comment:       r.Comment,
			SellerReply:   r.SellerReply,
			SellerReplyAt: r.SellerReplyAt,
			CreatedAt:     r.CreatedAt,
			CanEdit:       canEdit,
		}
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)
	return items, meta, nil
}

func (u *reviewUsecase) AdminFindAllReviews(ctx context.Context, filter repository.ReviewFilter, pq pagination.Query) ([]dto.AdminReviewItemResponse, pagination.Meta, error) {
	reviews, total, err := u.reviewRepo.FindAllAdmin(ctx, filter, pq)
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed fetching admin reviews: %w", err)
	}

	items := make([]dto.AdminReviewItemResponse, len(reviews))
	for i, r := range reviews {
		bookTitle := ""
		var bookCover *string
		if r.Book != nil {
			bookTitle = r.Book.Title
			bookCover = r.Book.CoverImage
		}

		invoiceNum := ""
		if r.Order != nil {
			invoiceNum = r.Order.InvoiceNumber
		}

		userName := ""
		userEmail := ""
		if r.User != nil {
			userName = r.User.Name
			userEmail = r.User.Email
		}

		items[i] = dto.AdminReviewItemResponse{
			ID:            r.ID,
			BookID:        r.BookID,
			BookTitle:     bookTitle,
			BookCover:     bookCover,
			OrderID:       r.OrderID,
			InvoiceNumber: invoiceNum,
			UserID:        r.UserID,
			UserName:      userName,
			UserEmail:     userEmail,
			Rating:        r.Rating,
			Comment:       r.Comment,
			IsHidden:      r.IsHidden,
			HiddenReason:  r.HiddenReason,
			SellerReply:   r.SellerReply,
			SellerReplyAt: r.SellerReplyAt,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		}
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)
	return items, meta, nil
}

func (u *reviewUsecase) AdminModerateReview(ctx context.Context, reviewID uint, req dto.AdminModerateReviewRequest) error {
	review, err := u.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}

	review.IsHidden = req.IsHidden
	if req.IsHidden {
		trimmed := strings.TrimSpace(req.Reason)
		if trimmed != "" {
			review.HiddenReason = &trimmed
		} else {
			defaultReason := "Moderated by administrator"
			review.HiddenReason = &defaultReason
		}
	} else {
		review.HiddenReason = nil
	}
	review.UpdatedAt = time.Now()

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := u.reviewRepo.Update(txCtx, review); err != nil {
			return err
		}
		return u.reviewRepo.RecalculateBookRating(txCtx, review.BookID)
	})

	if err != nil {
		return err
	}

	u.invalidateBookCache(ctx)
	return nil
}

func (u *reviewUsecase) AdminReplyReview(ctx context.Context, reviewID uint, req dto.AdminReplyReviewRequest) error {
	trimmed := strings.TrimSpace(req.Reply)
	if trimmed == "" {
		return errors.New("reply message cannot be empty")
	}

	review, err := u.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}

	now := time.Now()
	review.SellerReply = &trimmed
	review.SellerReplyAt = &now
	review.UpdatedAt = now

	return u.reviewRepo.Update(ctx, review)
}

func (u *reviewUsecase) AdminDeleteReview(ctx context.Context, reviewID uint) error {
	review, err := u.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}

	bookID := review.BookID

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := u.reviewRepo.Delete(txCtx, reviewID); err != nil {
			return err
		}
		return u.reviewRepo.RecalculateBookRating(txCtx, bookID)
	})

	if err != nil {
		return err
	}

	u.invalidateBookCache(ctx)
	return nil
}
