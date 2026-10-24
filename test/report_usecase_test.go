package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReportUsecase_GetOrderReport(t *testing.T) {
	t.Run("success calculates report summary and top books", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewReportUsecase(mockRepo)

		orderTime := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
		orders := []entity.Order{
			{
				ID:            1,
				UserID:        10,
				InvoiceNumber: "INV-20261015-0001",
				Status:        entity.OrderStatusPaid,
				ItemsSubtotal: 200000,
				ShippingCost:  20000,
				TotalAmount:   220000,
				RecipientName: "Alice",
				User: &entity.User{
					Name:  "Alice",
					Email: "alice@example.com",
				},
				Items: []entity.OrderItem{
					{
						BookID:   1,
						Title:    "Go Programming",
						Quantity: 2,
						Subtotal: 200000,
					},
				},
				CreatedAt: orderTime,
			},
			{
				ID:            2,
				UserID:        11,
				InvoiceNumber: "INV-20261015-0002",
				Status:        entity.OrderStatusCancelled,
				ItemsSubtotal: 100000,
				TotalAmount:   100000,
				RecipientName: "Bob",
				CreatedAt:     orderTime,
			},
		}

		filter := dto.OrderReportFilterRequest{
			StartDate: "2026-10-01",
			EndDate:   "2026-10-31",
			Status:    "all",
		}

		mockRepo.On("GetOrdersForReport", mock.Anything, mock.Anything, mock.Anything, "all").
			Return(orders, nil).Once()

		res, err := uc.GetOrderReport(context.Background(), filter)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 2, res.Summary.TotalOrders)
		assert.Equal(t, 1, res.Summary.PaidOrders)
		assert.Equal(t, 1, res.Summary.CancelledOrders)
		assert.Equal(t, 200000, res.Summary.GrossRevenue)
		assert.Equal(t, 220000, res.Summary.NetRevenue)
		assert.Equal(t, 2, res.Summary.TotalItemsSold)
		assert.Equal(t, 220000.0, res.Summary.AverageOrderValue)

		// Top books
		assert.Len(t, res.TopBooks, 1)
		assert.Equal(t, uint(1), res.TopBooks[0].BookID)
		assert.Equal(t, 2, res.TopBooks[0].QuantitySold)
		assert.Equal(t, 200000, res.TopBooks[0].TotalRevenue)

		mockRepo.AssertExpectations(t)
	})

	t.Run("success generates sales report pdf", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewReportUsecase(mockRepo)

		orderTime := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
		orders := []entity.Order{
			{
				ID:            1,
				UserID:        10,
				InvoiceNumber: "INV-20261015-0001",
				Status:        entity.OrderStatusPaid,
				ItemsSubtotal: 200000,
				ShippingCost:  20000,
				TotalAmount:   220000,
				RecipientName: "Alice",
				Items: []entity.OrderItem{
					{
						BookID:   1,
						Title:    "Go Programming",
						Quantity: 2,
						Subtotal: 200000,
					},
				},
				CreatedAt: orderTime,
			},
		}

		mockRepo.On("GetOrdersForReport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(orders, nil).Once()

		pdfBytes, filename, err := uc.GetOrderReportPDF(context.Background(), dto.OrderReportFilterRequest{})
		assert.NoError(t, err)
		assert.NotEmpty(t, pdfBytes)
		assert.Contains(t, filename, "bookstore-sales-report-")
		assert.True(t, len(pdfBytes) > 100)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success handles waiting_payment_orders, zero-filled daily trends, and pagination", func(t *testing.T) {
		mockRepo := new(MockOrderRepoForOrderUC)
		uc := usecase.NewReportUsecase(mockRepo)

		orderTime := time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC) // 10:00 AM WIB
		orders := []entity.Order{
			{
				ID:            1,
				InvoiceNumber: "INV-001",
				Status:        entity.OrderStatusWaitingPayment,
				ItemsSubtotal: 150000,
				TotalAmount:   150000,
				CreatedAt:     orderTime,
			},
			{
				ID:            2,
				InvoiceNumber: "INV-002",
				Status:        entity.OrderStatusPaid,
				ItemsSubtotal: 200000,
				TotalAmount:   200000,
				CreatedAt:     orderTime,
			},
			{
				ID:            3,
				InvoiceNumber: "INV-003",
				Status:        entity.OrderStatusCompleted,
				ItemsSubtotal: 300000,
				TotalAmount:   300000,
				CreatedAt:     orderTime,
			},
		}

		mockRepo.On("GetOrdersForReport", mock.Anything, mock.Anything, mock.Anything, "all").
			Return(orders, nil).Once()

		filter := dto.OrderReportFilterRequest{
			StartDate: "2026-10-18",
			EndDate:   "2026-10-20",
			Status:    "all",
			Page:      1,
			PerPage:   2,
		}

		res, err := uc.GetOrderReport(context.Background(), filter)
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Check waiting payment orders count
		assert.Equal(t, 3, res.Summary.TotalOrders)
		assert.Equal(t, 1, res.Summary.WaitingPaymentOrders)
		assert.Equal(t, 2, res.Summary.PaidOrders)

		// Check zero-filled daily trends: 18, 19, 20 (3 days)
		assert.Len(t, res.DailyTrends, 3)
		assert.Equal(t, "2026-10-18", res.DailyTrends[0].Date)
		assert.Equal(t, 0, res.DailyTrends[0].TotalOrders)
		assert.Equal(t, "2026-10-19", res.DailyTrends[1].Date)
		assert.Equal(t, 0, res.DailyTrends[1].TotalOrders)
		assert.Equal(t, "2026-10-20", res.DailyTrends[2].Date)
		assert.Equal(t, 2, res.DailyTrends[2].TotalOrders) // 2 revenue-eligible orders on Oct 20

		// Check pagination: 3 total, page 1 with per_page 2 returns 2 items
		assert.NotNil(t, res.Pagination)
		assert.Equal(t, 1, res.Pagination.CurrentPage)
		assert.Equal(t, 2, res.Pagination.PerPage)
		assert.Equal(t, int64(3), res.Pagination.TotalItems)
		assert.Equal(t, 2, res.Pagination.TotalPages)
		assert.Len(t, res.Orders, 2)
		assert.Equal(t, "INV-001", res.Orders[0].InvoiceNumber)
		assert.Equal(t, "INV-002", res.Orders[1].InvoiceNumber)

		mockRepo.AssertExpectations(t)
	})
}
