package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/pdf"
)

type ReportUsecase interface {
	GetOrderReport(ctx context.Context, filter dto.OrderReportFilterRequest) (*dto.OrderReportResponse, error)
	GetOrderReportPDF(ctx context.Context, filter dto.OrderReportFilterRequest) ([]byte, string, error)
}

type reportUsecase struct {
	orderRepo repository.OrderRepository
}

func NewReportUsecase(orderRepo repository.OrderRepository) ReportUsecase {
	return &reportUsecase{orderRepo: orderRepo}
}

func (u *reportUsecase) GetOrderReport(ctx context.Context, filter dto.OrderReportFilterRequest) (*dto.OrderReportResponse, error) {
	var startDate *time.Time
	var endDate *time.Time

	if filter.StartDate != "" {
		if t, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			startDate = &utc
		}
	}
	if filter.EndDate != "" {
		if t, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			utc := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
			endDate = &utc
		}
	}

	orders, err := u.orderRepo.GetOrdersForReport(ctx, startDate, endDate, filter.Status)
	if err != nil {
		return nil, fmt.Errorf("failed fetching report orders: %w", err)
	}

	var (
		totalOrders          = len(orders)
		waitingPaymentOrders = 0
		paidOrders           = 0
		completedOrders      = 0
		cancelledOrders      = 0
		returnedOrders       = 0
		grossRevenue         = 0
		netRevenue           = 0
		totalDiscounts       = 0
		totalItemsSold       = 0
	)

	bookStats := make(map[uint]*dto.TopSellingBookReport)
	trendStats := make(map[string]*dto.DailySalesTrend)

	// Pre-fill daily trends with zero values:
	// - If date range <= 30 days: zero-fill every day continuously for smooth charts.
	// - If date range > 30 days: zero-fill only the last 7 days of the period to keep payload compact,
	//   while any earlier dates with transactions are added dynamically.
	now := time.Now()
	if startDate != nil && endDate != nil {
		curr := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
		endLimit := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)
		daysDiff := int(endLimit.Sub(curr).Hours()/24) + 1
		if daysDiff <= 30 {
			for !curr.After(endLimit) {
				key := curr.Format("2006-01-02")
				trendStats[key] = &dto.DailySalesTrend{
					Date:         key,
					TotalOrders:  0,
					TotalRevenue: 0,
				}
				curr = curr.AddDate(0, 0, 1)
			}
		} else {
			recentStart := endLimit.AddDate(0, 0, -6)
			currRecent := recentStart
			for !currRecent.After(endLimit) {
				key := currRecent.Format("2006-01-02")
				trendStats[key] = &dto.DailySalesTrend{
					Date:         key,
					TotalOrders:  0,
					TotalRevenue: 0,
				}
				currRecent = currRecent.AddDate(0, 0, 1)
			}
		}
	} else if startDate != nil && endDate == nil {
		curr := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
		endLimit := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		daysDiff := int(endLimit.Sub(curr).Hours()/24) + 1
		if daysDiff <= 30 {
			for !curr.After(endLimit) {
				key := curr.Format("2006-01-02")
				trendStats[key] = &dto.DailySalesTrend{
					Date:         key,
					TotalOrders:  0,
					TotalRevenue: 0,
				}
				curr = curr.AddDate(0, 0, 1)
			}
		} else {
			recentStart := endLimit.AddDate(0, 0, -6)
			currRecent := recentStart
			for !currRecent.After(endLimit) {
				key := currRecent.Format("2006-01-02")
				trendStats[key] = &dto.DailySalesTrend{
					Date:         key,
					TotalOrders:  0,
					TotalRevenue: 0,
				}
				currRecent = currRecent.AddDate(0, 0, 1)
			}
		}
	} else if startDate == nil && endDate != nil {
		curr := endDate.AddDate(0, 0, -6)
		curr = time.Date(curr.Year(), curr.Month(), curr.Day(), 0, 0, 0, 0, time.UTC)
		endLimit := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)
		for !curr.After(endLimit) {
			key := curr.Format("2006-01-02")
			trendStats[key] = &dto.DailySalesTrend{
				Date:         key,
				TotalOrders:  0,
				TotalRevenue: 0,
			}
			curr = curr.AddDate(0, 0, 1)
		}
	} else {
		// Default: past 7 days up to today
		for i := 6; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			key := d.Format("2006-01-02")
			trendStats[key] = &dto.DailySalesTrend{
				Date:         key,
				TotalOrders:  0,
				TotalRevenue: 0,
			}
		}
	}

	allOrderItemsList := make([]dto.OrderReportItem, len(orders))

	for i, o := range orders {
		orderQty := 0
		for _, item := range o.Items {
			orderQty += item.Quantity
		}

		isRevenueEligible := o.Status == entity.OrderStatusPaid ||
			o.Status == entity.OrderStatusProcessing ||
			o.Status == entity.OrderStatusShipped ||
			o.Status == entity.OrderStatusCompleted

		switch o.Status {
		case entity.OrderStatusWaitingPayment:
			waitingPaymentOrders++
		case entity.OrderStatusPaid:
			paidOrders++
		case entity.OrderStatusCompleted:
			completedOrders++
			paidOrders++
		case entity.OrderStatusProcessing, entity.OrderStatusShipped:
			paidOrders++
		case entity.OrderStatusCancelled:
			cancelledOrders++
		case entity.OrderStatusReturned:
			returnedOrders++
		}

		if isRevenueEligible {
			grossRevenue += o.ItemsSubtotal
			netRevenue += o.TotalAmount
			totalDiscounts += o.DiscountAmount
			totalItemsSold += orderQty

			// Aggregate book sales
			for _, item := range o.Items {
				b, exists := bookStats[item.BookID]
				if !exists {
					cover := ""
					if item.CoverImage != nil {
						cover = *item.CoverImage
					}
					b = &dto.TopSellingBookReport{
						BookID:       item.BookID,
						Title:        item.Title,
						CoverImage:   cover,
						QuantitySold: 0,
						TotalRevenue: 0,
					}
					bookStats[item.BookID] = b
				}
				b.QuantitySold += item.Quantity
				b.TotalRevenue += item.Subtotal
			}

			// Aggregate daily trend
			dateKey := o.CreatedAt.Format("2006-01-02")
			trend, exists := trendStats[dateKey]
			if !exists {
				trend = &dto.DailySalesTrend{
					Date:         dateKey,
					TotalOrders:  0,
					TotalRevenue: 0,
				}
				trendStats[dateKey] = trend
			}
			trend.TotalOrders++
			trend.TotalRevenue += o.TotalAmount
		}

		custName := o.RecipientName
		custEmail := ""
		if o.User != nil {
			custName = o.User.Name
			custEmail = o.User.Email
		}

		payMethod := "Midtrans"
		if o.Payment != nil {
			if o.Payment.PaymentMethod != nil {
				payMethod = o.Payment.PaymentMethod.Name
			} else {
				payMethod = strings.ToUpper(strings.ReplaceAll(o.Payment.PaymentType, "_", " "))
				if o.Payment.Bank != nil && *o.Payment.Bank != "" {
					payMethod += " (" + strings.ToUpper(*o.Payment.Bank) + ")"
				} else if o.Payment.Store != nil && *o.Payment.Store != "" {
					payMethod += " (" + strings.ToUpper(*o.Payment.Store) + ")"
				}
			}
		}

		paidAt := o.PaidAt
		if paidAt == nil && o.Payment != nil {
			paidAt = o.Payment.PaidAt
		}

		allOrderItemsList[i] = dto.OrderReportItem{
			ID:             o.ID,
			InvoiceNumber:  o.InvoiceNumber,
			CustomerName:   custName,
			CustomerEmail:  custEmail,
			Status:         string(o.Status),
			TotalAmount:    o.TotalAmount,
			ItemsSubtotal:  o.ItemsSubtotal,
			ShippingCost:   o.ShippingCost,
			DiscountAmount: o.DiscountAmount,
			AdminFee:       o.AdminFee,
			PaymentMethod:  payMethod,
			TotalQuantity:  orderQty,
			CreatedAt:      o.CreatedAt,
			PaidAt:         paidAt,
		}
	}

	// Calculate Average Order Value (AOV)
	var aov float64
	if paidOrders > 0 {
		aov = float64(netRevenue) / float64(paidOrders)
	}

	// Sort Top Selling Books by Quantity Sold DESC
	topBooks := make([]dto.TopSellingBookReport, 0, len(bookStats))
	for _, b := range bookStats {
		topBooks = append(topBooks, *b)
	}
	sort.Slice(topBooks, func(i, j int) bool {
		if topBooks[i].QuantitySold == topBooks[j].QuantitySold {
			return topBooks[i].TotalRevenue > topBooks[j].TotalRevenue
		}
		return topBooks[i].QuantitySold > topBooks[j].QuantitySold
	})

	// Sort Daily Trends by Date ASC
	dailyTrends := make([]dto.DailySalesTrend, 0, len(trendStats))
	for _, t := range trendStats {
		dailyTrends = append(dailyTrends, *t)
	}
	sort.Slice(dailyTrends, func(i, j int) bool {
		return dailyTrends[i].Date < dailyTrends[j].Date
	})

	periodStartStr := "All Time"
	periodEndStr := "Today"
	if startDate != nil {
		periodStartStr = startDate.Format("02 Jan 2006")
	} else if len(orders) > 0 {
		periodStartStr = orders[0].CreatedAt.Format("02 Jan 2006")
	}
	if endDate != nil {
		periodEndStr = endDate.Format("02 Jan 2006")
	} else if len(orders) > 0 {
		periodEndStr = orders[len(orders)-1].CreatedAt.Format("02 Jan 2006")
	}

	// Orders pagination
	var orderItemsList []dto.OrderReportItem
	var paginationMeta *pagination.Meta

	if filter.Page > 0 && filter.PerPage > 0 {
		totalCount := int64(len(allOrderItemsList))
		meta := pagination.BuildMeta(totalCount, filter.Page, filter.PerPage)
		paginationMeta = &meta

		offset := (filter.Page - 1) * filter.PerPage
		if offset >= len(allOrderItemsList) {
			orderItemsList = []dto.OrderReportItem{}
		} else {
			end := offset + filter.PerPage
			if end > len(allOrderItemsList) {
				end = len(allOrderItemsList)
			}
			orderItemsList = allOrderItemsList[offset:end]
		}
	} else {
		orderItemsList = allOrderItemsList
	}

	return &dto.OrderReportResponse{
		PeriodStart: periodStartStr,
		PeriodEnd:   periodEndStr,
		GeneratedAt: time.Now(),
		Summary: dto.OrderReportSummary{
			TotalOrders:          totalOrders,
			WaitingPaymentOrders: waitingPaymentOrders,
			PaidOrders:           paidOrders,
			CompletedOrders:      completedOrders,
			CancelledOrders:      cancelledOrders,
			ReturnedOrders:       returnedOrders,
			GrossRevenue:         grossRevenue,
			NetRevenue:           netRevenue,
			TotalDiscounts:       totalDiscounts,
			TotalItemsSold:       totalItemsSold,
			AverageOrderValue:    aov,
		},
		TopBooks:    topBooks,
		DailyTrends: dailyTrends,
		Orders:      orderItemsList,
		Pagination:  paginationMeta,
	}, nil
}

func (u *reportUsecase) GetOrderReportPDF(ctx context.Context, filter dto.OrderReportFilterRequest) ([]byte, string, error) {
	// PDF always exports all orders in the filtered period
	pdfFilter := filter
	pdfFilter.Page = 0
	pdfFilter.PerPage = 0

	report, err := u.GetOrderReport(ctx, pdfFilter)
	if err != nil {
		return nil, "", err
	}

	pdfBytes, err := pdf.GenerateSalesReportPDF(*report)
	if err != nil {
		return nil, "", fmt.Errorf("failed generating sales report pdf: %w", err)
	}

	filename := fmt.Sprintf("bookstore-sales-report-%s.pdf", time.Now().Format("20060102-150405"))
	return pdfBytes, filename, nil
}
