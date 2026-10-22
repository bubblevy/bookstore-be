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
}
