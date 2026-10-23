package dto

import "time"

type OrderReportFilterRequest struct {
	StartDate string `query:"start_date"` // format YYYY-MM-DD
	EndDate   string `query:"end_date"`   // format YYYY-MM-DD
	Status    string `query:"status"`     // "all", "paid", "completed", etc.
}

type TopSellingBookReport struct {
	BookID       uint   `json:"book_id"`
	Title        string `json:"title"`
	CoverImage   string `json:"cover_image,omitempty"`
	QuantitySold int    `json:"quantity_sold"`
	TotalRevenue int    `json:"total_revenue"`
}

type DailySalesTrend struct {
	Date         string `json:"date"` // YYYY-MM-DD
	TotalOrders  int    `json:"total_orders"`
	TotalRevenue int    `json:"total_revenue"`
}

type OrderReportSummary struct {
	TotalOrders       int     `json:"total_orders"`
	PaidOrders        int     `json:"paid_orders"`
	CompletedOrders   int     `json:"completed_orders"`
	CancelledOrders   int     `json:"cancelled_orders"`
	ReturnedOrders    int     `json:"returned_orders"`
	GrossRevenue      int     `json:"gross_revenue"`
	NetRevenue        int     `json:"net_revenue"`
	TotalDiscounts    int     `json:"total_discounts"`
	TotalItemsSold    int     `json:"total_items_sold"`
	AverageOrderValue float64 `json:"average_order_value"`
}

type OrderReportItem struct {
	ID             uint       `json:"id"`
	InvoiceNumber  string     `json:"invoice_number"`
	CustomerName   string     `json:"customer_name"`
	CustomerEmail  string     `json:"customer_email"`
	Status         string     `json:"status"`
	TotalAmount    int        `json:"total_amount"`
	ItemsSubtotal  int        `json:"items_subtotal"`
	ShippingCost   int        `json:"shipping_cost"`
	DiscountAmount int        `json:"discount_amount"`
	AdminFee       int        `json:"admin_fee"`
	PaymentMethod  string     `json:"payment_method"`
	TotalQuantity  int        `json:"total_quantity"`
	CreatedAt      time.Time  `json:"created_at"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
}

type OrderReportResponse struct {
	PeriodStart string                 `json:"period_start"`
	PeriodEnd   string                 `json:"period_end"`
	GeneratedAt time.Time              `json:"generated_at"`
	Summary     OrderReportSummary     `json:"summary"`
	TopBooks    []TopSellingBookReport `json:"top_books"`
	DailyTrends []DailySalesTrend      `json:"daily_trends"`
	Orders      []OrderReportItem      `json:"orders"`
}
