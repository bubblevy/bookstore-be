package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockReviewRepository struct {
	mock.Mock
}

func (m *MockReviewRepository) Create(ctx context.Context, review *entity.Review) error {
	return m.Called(ctx, review).Error(0)
}

func (m *MockReviewRepository) CreateBatch(ctx context.Context, reviews []entity.Review) error {
	return m.Called(ctx, reviews).Error(0)
}

func (m *MockReviewRepository) Update(ctx context.Context, review *entity.Review) error {
	return m.Called(ctx, review).Error(0)
}

func (m *MockReviewRepository) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockReviewRepository) FindByID(ctx context.Context, id uint) (*entity.Review, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*entity.Review), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewRepository) FindByOrderAndBook(ctx context.Context, orderID, bookID uint) (*entity.Review, error) {
	args := m.Called(ctx, orderID, bookID)
	if res := args.Get(0); res != nil {
		return res.(*entity.Review), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewRepository) FindByOrderID(ctx context.Context, orderID uint) ([]entity.Review, error) {
	args := m.Called(ctx, orderID)
	if res := args.Get(0); res != nil {
		return res.([]entity.Review), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewRepository) FindBookReviews(ctx context.Context, bookID uint, filter repository.ReviewFilter, pq pagination.Query) ([]entity.Review, int64, error) {
	args := m.Called(ctx, bookID, filter, pq)
	if res := args.Get(0); res != nil {
		return res.([]entity.Review), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockReviewRepository) GetBookReviewStats(ctx context.Context, bookID uint) (*repository.ReviewStats, error) {
	args := m.Called(ctx, bookID)
	if res := args.Get(0); res != nil {
		return res.(*repository.ReviewStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReviewRepository) FindUserReviews(ctx context.Context, userID uint, pq pagination.Query) ([]entity.Review, int64, error) {
	args := m.Called(ctx, userID, pq)
	if res := args.Get(0); res != nil {
		return res.([]entity.Review), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockReviewRepository) FindAllAdmin(ctx context.Context, filter repository.ReviewFilter, pq pagination.Query) ([]entity.Review, int64, error) {
	args := m.Called(ctx, filter, pq)
	if res := args.Get(0); res != nil {
		return res.([]entity.Review), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockReviewRepository) RecalculateBookRating(ctx context.Context, bookID uint) error {
	return m.Called(ctx, bookID).Error(0)
}

type MockTxManager struct{}

func (m *MockTxManager) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

func TestReviewUsecase_GetBookReviews_Breakdown(t *testing.T) {
	mockReviewRepo := new(MockReviewRepository)
	mockOrderRepo := new(MockOrderRepoForOrderUC)
	mockBookRepo := new(MockBookRepository)
	txManager := &MockTxManager{}

	uc := usecase.NewReviewUsecase(mockReviewRepo, mockOrderRepo, mockBookRepo, txManager, nil)

	ctx := context.Background()
	bookID := uint(10)
	book := &entity.Book{
		Title: "Golang in Action",
		Slug:  "golang-in-action",
	}
	book.ID = bookID

	mockBookRepo.On("FindBySlug", ctx, "golang-in-action").Return(book, nil)

	stats := &repository.ReviewStats{
		AverageRating:    4.8,
		TotalReviews:     10,
		Star5Count:       8,
		Star4Count:       2,
		Star3Count:       0,
		Star2Count:       0,
		Star1Count:       0,
		WithCommentCount: 5,
	}
	mockReviewRepo.On("GetBookReviewStats", ctx, bookID).Return(stats, nil)

	comment := "Buku mantap sekali!"
	reviews := []entity.Review{
		{
			ID:        1,
			Rating:    5,
			Comment:   &comment,
			User:      &entity.User{Name: "Budi Santoso"},
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			Rating:    4,
			User:      &entity.User{Name: "Alice"},
			CreatedAt: time.Now(),
		},
	}
	pq := pagination.Query{Page: 1, PerPage: 10}
	filter := repository.ReviewFilter{SortBy: "newest"}
	mockReviewRepo.On("FindBookReviews", ctx, bookID, filter, pq).Return(reviews, int64(10), nil)

	resp, err := uc.GetBookReviews(ctx, "golang-in-action", filter, pq)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 4.8, resp.Breakdown.AverageRating)
	assert.Equal(t, int64(10), resp.Breakdown.TotalReviews)
	assert.Equal(t, int64(8), resp.Breakdown.Star5Count)
	assert.Equal(t, int64(2), resp.Breakdown.Star4Count)

	assert.Len(t, resp.Reviews, 2)
	assert.Equal(t, "B****i S****o", resp.Reviews[0].UserName)
	assert.Equal(t, "A****e", resp.Reviews[1].UserName)
}

func TestReviewUsecase_CheckOrderReviewEligibility(t *testing.T) {
	mockReviewRepo := new(MockReviewRepository)
	mockOrderRepo := new(MockOrderRepoForOrderUC)
	mockBookRepo := new(MockBookRepository)
	txManager := &MockTxManager{}

	uc := usecase.NewReviewUsecase(mockReviewRepo, mockOrderRepo, mockBookRepo, txManager, nil)

	ctx := context.Background()
	userID := uint(1)
	invoice := "INV-12345"

	t.Run("Eligible within 7 days", func(t *testing.T) {
		completedAt := time.Now().Add(-2 * 24 * time.Hour) // 2 days ago
		order := &entity.Order{
			ID:            100,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusCompleted,
			CompletedAt:   &completedAt,
			Items: []entity.OrderItem{
				{BookID: 10, Title: "Book A", Price: 50000},
				{BookID: 20, Title: "Book B", Price: 75000},
			},
		}

		mockOrderRepo.On("FindOrderDetailByInvoice", ctx, userID, invoice).Return(order, nil).Once()

		existingReviews := []entity.Review{
			{ID: 1, BookID: 10, Rating: 5},
		}
		mockReviewRepo.On("FindByOrderID", ctx, uint(100)).Return(existingReviews, nil).Once()

		res, err := uc.CheckOrderReviewEligibility(ctx, userID, invoice)
		assert.NoError(t, err)
		assert.True(t, res.IsEligible)
		assert.False(t, res.IsExpired)
		assert.GreaterOrEqual(t, res.RemainingDays, 4)
		assert.Greater(t, res.RemainingHours, 90)
		assert.Len(t, res.Items, 2)
		assert.True(t, res.Items[0].IsReviewed)
		assert.False(t, res.Items[1].IsReviewed)
	})

	t.Run("Expired after 7 days", func(t *testing.T) {
		completedAt := time.Now().Add(-8 * 24 * time.Hour) // 8 days ago
		order := &entity.Order{
			ID:            101,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusCompleted,
			CompletedAt:   &completedAt,
		}

		mockOrderRepo.On("FindOrderDetailByInvoice", ctx, userID, invoice).Return(order, nil).Once()
		mockReviewRepo.On("FindByOrderID", ctx, uint(101)).Return([]entity.Review{}, nil).Once()

		res, err := uc.CheckOrderReviewEligibility(ctx, userID, invoice)
		assert.NoError(t, err)
		assert.False(t, res.IsEligible)
		assert.True(t, res.IsExpired)
	})

	t.Run("Ineligible if order not completed", func(t *testing.T) {
		order := &entity.Order{
			ID:            102,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusShipped,
		}

		mockOrderRepo.On("FindOrderDetailByInvoice", ctx, userID, invoice).Return(order, nil).Once()
		mockReviewRepo.On("FindByOrderID", ctx, uint(102)).Return([]entity.Review{}, nil).Once()

		res, err := uc.CheckOrderReviewEligibility(ctx, userID, invoice)
		assert.NoError(t, err)
		assert.False(t, res.IsEligible)
	})
}

func TestReviewUsecase_SubmitOrderReviews(t *testing.T) {
	mockReviewRepo := new(MockReviewRepository)
	mockOrderRepo := new(MockOrderRepoForOrderUC)
	mockBookRepo := new(MockBookRepository)
	txManager := &MockTxManager{}

	uc := usecase.NewReviewUsecase(mockReviewRepo, mockOrderRepo, mockBookRepo, txManager, nil)

	ctx := context.Background()
	userID := uint(1)
	invoice := "INV-SUBMIT-1"

	t.Run("Success submission within 7 days", func(t *testing.T) {
		completedAt := time.Now().Add(-1 * 24 * time.Hour)
		order := &entity.Order{
			ID:            200,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusCompleted,
			CompletedAt:   &completedAt,
			Items: []entity.OrderItem{
				{BookID: 10, Title: "Book 1"},
			},
		}

		mockOrderRepo.On("FindOrderDetailByInvoice", ctx, userID, invoice).Return(order, nil).Once()
		mockReviewRepo.On("FindByOrderAndBook", ctx, uint(200), uint(10)).Return(nil, entity.ErrReviewNotFound).Once()
		mockReviewRepo.On("Create", ctx, mock.AnythingOfType("*entity.Review")).Return(nil).Once()
		mockReviewRepo.On("RecalculateBookRating", ctx, uint(10)).Return(nil).Once()

		req := dto.SubmitOrderReviewsRequest{
			Reviews: []dto.CreateReviewItemRequest{
				{
					BookID:  10,
					Rating:  5,
					Comment: "Sangat puas!",
				},
			},
		}

		res, err := uc.SubmitOrderReviews(ctx, userID, invoice, req)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, 5, res[0].Rating)
	})

	t.Run("Fail when rating expired", func(t *testing.T) {
		completedAt := time.Now().Add(-8 * 24 * time.Hour)
		order := &entity.Order{
			ID:            201,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusCompleted,
			CompletedAt:   &completedAt,
			Items: []entity.OrderItem{
				{BookID: 10, Title: "Book 1"},
			},
		}

		mockOrderRepo.On("FindOrderDetailByInvoice", ctx, userID, invoice).Return(order, nil).Once()

		req := dto.SubmitOrderReviewsRequest{
			Reviews: []dto.CreateReviewItemRequest{
				{BookID: 10, Rating: 5},
			},
		}

		res, err := uc.SubmitOrderReviews(ctx, userID, invoice, req)
		assert.ErrorIs(t, err, entity.ErrRatingPeriodExpired)
		assert.Nil(t, res)
	})

	t.Run("Fail when order not completed", func(t *testing.T) {
		order := &entity.Order{
			ID:            202,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusPaid,
		}

		mockOrderRepo.On("FindOrderDetailByInvoice", ctx, userID, invoice).Return(order, nil).Once()

		req := dto.SubmitOrderReviewsRequest{
			Reviews: []dto.CreateReviewItemRequest{
				{BookID: 10, Rating: 5},
			},
		}

		res, err := uc.SubmitOrderReviews(ctx, userID, invoice, req)
		assert.ErrorIs(t, err, entity.ErrReviewNotEligible)
		assert.Nil(t, res)
	})

	t.Run("Fail when book not in order", func(t *testing.T) {
		completedAt := time.Now()
		order := &entity.Order{
			ID:            203,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusCompleted,
			CompletedAt:   &completedAt,
			Items: []entity.OrderItem{
				{BookID: 10, Title: "Book 1"},
			},
		}

		mockOrderRepo.On("FindOrderDetailByInvoice", ctx, userID, invoice).Return(order, nil).Once()

		req := dto.SubmitOrderReviewsRequest{
			Reviews: []dto.CreateReviewItemRequest{
				{BookID: 999, Rating: 5},
			},
		}

		res, err := uc.SubmitOrderReviews(ctx, userID, invoice, req)
		assert.ErrorIs(t, err, entity.ErrBookNotInOrder)
		assert.Nil(t, res)
	})
}

func TestReviewUsecase_AdminModerateAndReply(t *testing.T) {
	mockReviewRepo := new(MockReviewRepository)
	mockOrderRepo := new(MockOrderRepoForOrderUC)
	mockBookRepo := new(MockBookRepository)
	txManager := &MockTxManager{}

	uc := usecase.NewReviewUsecase(mockReviewRepo, mockOrderRepo, mockBookRepo, txManager, nil)
	ctx := context.Background()

	t.Run("Admin moderate visibility", func(t *testing.T) {
		review := &entity.Review{
			ID:     5,
			BookID: 10,
		}
		mockReviewRepo.On("FindByID", ctx, uint(5)).Return(review, nil).Once()
		mockReviewRepo.On("Update", ctx, mock.AnythingOfType("*entity.Review")).Return(nil).Once()
		mockReviewRepo.On("RecalculateBookRating", ctx, uint(10)).Return(nil).Once()

		req := dto.AdminModerateReviewRequest{
			IsHidden: true,
			Reason:   "Spam comment",
		}
		err := uc.AdminModerateReview(ctx, 5, req)
		assert.NoError(t, err)
		assert.True(t, review.IsHidden)
		assert.Equal(t, "Spam comment", *review.HiddenReason)
	})

	t.Run("Admin reply to review", func(t *testing.T) {
		review := &entity.Review{
			ID:     6,
			BookID: 10,
		}
		mockReviewRepo.On("FindByID", ctx, uint(6)).Return(review, nil).Once()
		mockReviewRepo.On("Update", ctx, mock.AnythingOfType("*entity.Review")).Return(nil).Once()

		req := dto.AdminReplyReviewRequest{
			Reply: "Terima kasih banyak atas pesanannya kak!",
		}
		err := uc.AdminReplyReview(ctx, 6, req)
		assert.NoError(t, err)
		assert.Equal(t, "Terima kasih banyak atas pesanannya kak!", *review.SellerReply)
		assert.NotNil(t, review.SellerReplyAt)
	})

	t.Run("Admin delete review", func(t *testing.T) {
		review := &entity.Review{
			ID:     7,
			BookID: 10,
		}
		mockReviewRepo.On("FindByID", ctx, uint(7)).Return(review, nil).Once()
		mockReviewRepo.On("Delete", ctx, uint(7)).Return(nil).Once()
		mockReviewRepo.On("RecalculateBookRating", ctx, uint(10)).Return(nil).Once()

		err := uc.AdminDeleteReview(ctx, 7)
		assert.NoError(t, err)
	})
}
