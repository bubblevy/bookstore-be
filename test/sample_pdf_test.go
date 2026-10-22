package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/pkg/pdf"
)

func TestGenerateSamplePDFs(t *testing.T) {
	report := dto.OrderReportResponse{
		PeriodStart: "01 Okt 2026",
		PeriodEnd:   "22 Okt 2026",
		GeneratedAt: time.Now(),
		Summary: dto.OrderReportSummary{
			TotalOrders:       28,
			PaidOrders:        24,
			CompletedOrders:   19,
			CancelledOrders:   3,
			ReturnedOrders:    1,
			GrossRevenue:      4500000,
			NetRevenue:        4850000,
			TotalDiscounts:    100000,
			TotalItemsSold:    65,
			AverageOrderValue: 202083.33,
		},
		TopBooks: []dto.TopSellingBookReport{
			{BookID: 1, Title: "Building Microservices with Go", QuantitySold: 25, TotalRevenue: 2250000},
			{BookID: 2, Title: "Designing Data-Intensive Applications", QuantitySold: 18, TotalRevenue: 1620000},
			{BookID: 3, Title: "Clean Architecture in Go", QuantitySold: 12, TotalRevenue: 960000},
		},
		Orders: []dto.OrderReportItem{
			{
				ID:            1,
				InvoiceNumber: "INV-20261021-000101",
				CustomerName:  "Budi Santoso",
				Status:        "paid",
				TotalAmount:   245000,
				ItemsSubtotal: 230000,
				ShippingCost:  20000,
				PaymentMethod: "BCA VA",
				TotalQuantity: 2,
				CreatedAt:     time.Now().Add(-2 * time.Hour),
			},
			{
				ID:            2,
				InvoiceNumber: "INV-20261021-000102",
				CustomerName:  "Siti Nurhaliza",
				Status:        "waiting_payment",
				TotalAmount:   180000,
				ItemsSubtotal: 160000,
				ShippingCost:  20000,
				PaymentMethod: "GOPAY",
				TotalQuantity: 1,
				CreatedAt:     time.Now().Add(-1 * time.Hour),
			},
		},
	}

	reportBytes, err := pdf.GenerateSalesReportPDF(report)
	if err != nil {
		t.Fatalf("failed to generate report pdf: %v", err)
	}

	order := &entity.Order{
		ID:               1,
		InvoiceNumber:    "INV-20261021-000101",
		Status:           entity.OrderStatusPaid,
		ItemsSubtotal:    230000,
		ShippingCost:     20000,
		DiscountAmount:   10000,
		AdminFee:         5000,
		TotalAmount:      245000,
		Courier:          "jne",
		CourierService:   "REG",
		TotalWeightGrams: 450,
		RecipientName:    "Budi Santoso",
		RecipientPhone:   "081298765432",
		RecipientAddress: "Jl. Jenderal Sudirman Kav. 52-53",
		CityName:         "Jakarta Selatan",
		ProvinceName:     "DKI Jakarta",
		PostalCode:       "12190",
		PaymentDeadline:  time.Now().Add(24 * time.Hour),
		Items: []entity.OrderItem{
			{
				BookID:      1,
				Title:       "Building Microservices with Go",
				Price:       115000,
				Quantity:    2,
				WeightGrams: 450,
				Subtotal:    230000,
			},
		},
		CreatedAt: time.Now(),
	}

	bank := "bca"
	va := "123456789012"
	order.Payment = &entity.Payment{
		PaymentType: "bank_transfer",
		Bank:        &bank,
		VANumber:    &va,
		Status:      entity.PaymentStatusSettlement,
	}

	invoiceBytes, err := pdf.GenerateInvoicePDF(order)
	if err != nil {
		t.Fatalf("failed to generate invoice pdf: %v", err)
	}

	artifactDir := `C:\Users\hp5cd\.gemini\antigravity-ide\brain\c5c3215f-586b-41ed-9e55-61277986d539\scratch`
	_ = os.MkdirAll(artifactDir, 0755)
	_ = os.WriteFile(filepath.Join(artifactDir, "sample_report.pdf"), reportBytes, 0644)
	_ = os.WriteFile(filepath.Join(artifactDir, "sample_invoice.pdf"), invoiceBytes, 0644)
}
