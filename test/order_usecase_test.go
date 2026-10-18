package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockOrderRepoForOrderUC struct {
	mock.Mock
}

func (m *MockOrderRepoForOrderUC) WithTx(tx *gorm.DB) repository.OrderRepository {
	return m
}

func (m *MockOrderRepoForOrderUC) CreateOrder(ctx context.Context, order *entity.Order, items []entity.OrderItem) error {
	return m.Called(ctx, order, items).Error(0)
}

func (m *MockOrderRepoForOrderUC) CreatePayment(ctx context.Context, p *entity.Payment) error {
	return m.Called(ctx, p).Error(0)
}

func (m *MockOrderRepoForOrderUC) FindByInvoice(ctx context.Context, invoice string) (*entity.Order, error) {
	args := m.Called(ctx, invoice)
	if res := args.Get(0); res != nil {
		return res.(*entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepoForOrderUC) FindByID(ctx context.Context, id uint) (*entity.Order, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepoForOrderUC) FindUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error) {
	args := m.Called(ctx, userID, statusFilter)
	if res := args.Get(0); res != nil {
		return res.([]entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepoForOrderUC) FindOrderDetailByInvoice(ctx context.Context, userID uint, invoice string) (*entity.Order, error) {
	args := m.Called(ctx, userID, invoice)
	if res := args.Get(0); res != nil {
		return res.(*entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepoForOrderUC) UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error {
	return m.Called(ctx, orderID, status).Error(0)
}

func (m *MockOrderRepoForOrderUC) UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error {
	return m.Called(ctx, orderID, status).Error(0)
}

func (m *MockOrderRepoForOrderUC) UpdatePaymentDetails(ctx context.Context, orderID uint, status entity.PaymentStatus, rawResponse *string, paidAt *time.Time) error {
	args := m.Called(ctx, orderID, status, rawResponse, paidAt)
	return args.Error(0)
}

func (m *MockOrderRepoForOrderUC) MarkAsReturned(ctx context.Context, orderID uint, reason string) error {
	return m.Called(ctx, orderID, reason).Error(0)
}

func TestOrderUsecase_GetUserOrders(t *testing.T) {
	userID := uint(1)

	t.Run("success returns user orders from repository", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewOrderUsecase(mockRepo)

		dummyOrders := []entity.Order{
			{
				ID:            1,
				UserID:        userID,
				InvoiceNumber: "INV-20261017-000101",
				Status:        entity.OrderStatusWaitingPayment,
				TotalAmount:   150000,
				CreatedAt:     time.Now(),
			},
		}

		mockRepo.On("FindUserOrders", mock.Anything, userID, "").Return(dummyOrders, nil).Once()

		res, err := uc.GetUserOrders(context.Background(), userID, "")
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "INV-20261017-000101", res[0].InvoiceNumber)
		mockRepo.AssertExpectations(t)
	})
}

func TestOrderUsecase_GetOrderDetail(t *testing.T) {
	userID := uint(1)
	invoice := "INV-20261017-000101"

	t.Run("success returns order detail when found", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewOrderUsecase(mockRepo)

		order := &entity.Order{
			ID:            1,
			UserID:        userID,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusWaitingPayment,
			TotalAmount:   150000,
		}

		mockRepo.On("FindOrderDetailByInvoice", mock.Anything, userID, invoice).Return(order, nil).Once()

		res, err := uc.GetOrderDetail(context.Background(), userID, invoice)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, invoice, res.InvoiceNumber)
		mockRepo.AssertExpectations(t)
	})

	t.Run("failed returns ErrOrderNotFound when repository returns error", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewOrderUsecase(mockRepo)

		mockRepo.On("FindOrderDetailByInvoice", mock.Anything, userID, "INV-NONE").
			Return(nil, errors.New("record not found")).Once()

		res, err := uc.GetOrderDetail(context.Background(), userID, "INV-NONE")
		assert.ErrorIs(t, err, entity.ErrOrderNotFound)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})
}
