package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
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

func (m *MockOrderRepoForOrderUC) UpdateOrderPaymentDeadline(ctx context.Context, orderID uint, deadline time.Time) error {
	return m.Called(ctx, orderID, deadline).Error(0)
}

func (m *MockOrderRepoForOrderUC) MarkAsReturned(ctx context.Context, orderID uint, reason string) error {
	return m.Called(ctx, orderID, reason).Error(0)
}

func (m *MockOrderRepoForOrderUC) FindAllOrders(ctx context.Context, filter repository.AdminOrderFilter, pq pagination.Query) ([]entity.Order, int64, error) {
	args := m.Called(ctx, filter, pq)
	if res := args.Get(0); res != nil {
		return res.([]entity.Order), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockOrderRepoForOrderUC) GetOrdersForReport(ctx context.Context, startDate, endDate *time.Time, status string) ([]entity.Order, error) {
	args := m.Called(ctx, startDate, endDate, status)
	if res := args.Get(0); res != nil {
		return res.([]entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
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

func TestOrderUsecase_AdminFindAllOrders(t *testing.T) {
	t.Run("success returns paginated admin orders with mapping", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewOrderUsecase(mockRepo)

		orders := []entity.Order{
			{
				ID:            1,
				UserID:        10,
				InvoiceNumber: "INV-20261021-000101",
				Status:        entity.OrderStatusPaid,
				TotalAmount:   200000,
				RecipientName: "John Doe",
				User: &entity.User{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				Items: []entity.OrderItem{
					{
						BookID:   1,
						Title:    "Golang Architecture",
						Quantity: 2,
					},
				},
				CreatedAt: time.Now(),
			},
		}

		filter := repository.AdminOrderFilter{Status: "paid"}
		pq := pagination.Query{Page: 1, PerPage: 10}

		mockRepo.On("FindAllOrders", mock.Anything, filter, pq).
			Return(orders, int64(1), nil).Once()

		items, meta, err := uc.AdminFindAllOrders(context.Background(), filter, pq)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "INV-20261021-000101", items[0].InvoiceNumber)
		assert.Equal(t, "John Doe", items[0].Customer.Name)
		assert.Equal(t, 2, items[0].TotalQuantity)
		assert.Equal(t, 200000, items[0].TotalAmount)
		assert.Equal(t, 1, meta.TotalPages)
		assert.Equal(t, int64(1), meta.TotalItems)
		mockRepo.AssertExpectations(t)
	})
}

func TestOrderUsecase_GetOrderInvoicePDF(t *testing.T) {
	invoice := "INV-20261021-000101"
	userID := uint(10)

	t.Run("success generates invoice pdf for order", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewOrderUsecase(mockRepo)

		order := &entity.Order{
			ID:               1,
			UserID:           userID,
			InvoiceNumber:    invoice,
			Status:           entity.OrderStatusPaid,
			ItemsSubtotal:    150000,
			ShippingCost:     15000,
			TotalAmount:      165000,
			Courier:          "jne",
			CourierService:   "REG",
			TotalWeightGrams: 300,
			RecipientName:    "Jane Doe",
			RecipientPhone:   "08123456789",
			RecipientAddress: "Jl. Sudirman No. 1",
			CityName:         "Jakarta Selatan",
			ProvinceName:     "DKI Jakarta",
			PostalCode:       "12190",
			PaymentDeadline:  time.Now().Add(3 * time.Hour),
			Items: []entity.OrderItem{
				{
					BookID:      1,
					Title:       "Clean Code",
					Price:       150000,
					Quantity:    1,
					WeightGrams: 300,
					Subtotal:    150000,
				},
			},
			CreatedAt: time.Now(),
		}

		mockRepo.On("FindOrderDetailByInvoice", mock.Anything, uint(0), invoice).
			Return(order, nil).Once()

		pdfBytes, filename, err := uc.GetOrderInvoicePDF(context.Background(), userID, invoice, true)
		assert.NoError(t, err)
		assert.NotEmpty(t, pdfBytes)
		assert.Equal(t, "invoice-"+invoice+".pdf", filename)
		assert.True(t, len(pdfBytes) > 100)
		mockRepo.AssertExpectations(t)
	})
}

func TestOrderUsecase_AdminGetOrderDetail(t *testing.T) {
	invoice := "INV-20261021-000101"

	t.Run("success returns admin order detail", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewOrderUsecase(mockRepo)

		order := &entity.Order{
			ID:            1,
			UserID:        10,
			InvoiceNumber: invoice,
			Status:        entity.OrderStatusPaid,
			TotalAmount:   200000,
			RecipientName: "John Doe",
			User: &entity.User{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			Items: []entity.OrderItem{
				{
					BookID:      1,
					Title:       "Clean Code",
					Price:       200000,
					Quantity:    1,
					WeightGrams: 300,
					Subtotal:    200000,
				},
			},
			CreatedAt: time.Now(),
		}

		mockRepo.On("FindOrderDetailByInvoice", mock.Anything, uint(0), invoice).
			Return(order, nil).Once()

		res, err := uc.AdminGetOrderDetail(context.Background(), invoice)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, invoice, res.InvoiceNumber)
		assert.Equal(t, "John Doe", res.Customer.Name)
		assert.Equal(t, "john@example.com", res.Customer.Email)
		assert.Equal(t, 200000, res.Pricing.TotalAmount)
		assert.Len(t, res.Items, 1)
		mockRepo.AssertExpectations(t)
	})
}

