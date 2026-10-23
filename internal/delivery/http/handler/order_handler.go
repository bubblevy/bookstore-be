package handler

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type OrderHandler struct {
	orderUsecase usecase.OrderUsecase
}

func NewOrderHandler(r fiber.Router, orderUC usecase.OrderUsecase, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &OrderHandler{orderUsecase: orderUC}

	// Order routes
	orders := r.Group("/orders", authMiddleware)
	orders.Get("/all", adminMiddleware, h.AdminFindAllOrders)
	orders.Get("/detail/:invoice_number", adminMiddleware, h.AdminGetOrderDetail)
	orders.Get("/", h.GetUserOrders)
	orders.Get("/:invoice_number/pdf", h.DownloadInvoicePDF)
	orders.Get("/:invoice_number", h.GetOrderDetail)
}

func (h *OrderHandler) getUserID(c fiber.Ctx) (uint, error) {
	val := c.Locals("user_id")
	if val == nil {
		return 0, fiber.ErrUnauthorized
	}
	switch v := val.(type) {
	case uint:
		return v, nil
	case int:
		return uint(v), nil
	case float64:
		return uint(v), nil
	default:
		return 0, fiber.ErrUnauthorized
	}
}

func (h *OrderHandler) GetUserOrders(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	statusFilter := strings.TrimSpace(c.Query("status"))
	orders, err := h.orderUsecase.GetUserOrders(c.Context(), userID, statusFilter)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	resp := make([]dto.OrderListItemResponse, len(orders))
	for i, o := range orders {
		var firstTitle string
		var firstCover *string
		totalQty := 0

		if len(o.Items) > 0 {
			firstTitle = o.Items[0].Title
			firstCover = o.Items[0].CoverImage
			for _, item := range o.Items {
				totalQty += item.Quantity
			}
		}

		extraCount := 0
		if len(o.Items) > 1 {
			extraCount = len(o.Items) - 1
		}

		resp[i] = dto.OrderListItemResponse{
			ID:             o.ID,
			InvoiceNumber:  o.InvoiceNumber,
			Status:         string(o.Status),
			TotalAmount:    o.TotalAmount,
			TotalQuantity:  totalQty,
			FirstItemTitle: firstTitle,
			FirstItemCover: firstCover,
			ExtraItemCount: extraCount,
			CreatedAt:      o.CreatedAt,
		}
	}

	return response.Success(c, fiber.StatusOK, "orders retrieved successfully", resp)
}

func (h *OrderHandler) GetOrderDetail(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	rawInvoice := c.Params("invoice_number")
	invoiceNumber, err := url.QueryUnescape(rawInvoice)
	if err != nil {
		invoiceNumber = rawInvoice
	}
	invoiceNumber = strings.TrimSpace(invoiceNumber)

	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	order, err := h.orderUsecase.GetOrderDetail(c.Context(), userID, invoiceNumber)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, "order not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	itemsResp := make([]dto.OrderItemResponse, len(order.Items))
	for i, it := range order.Items {
		itemsResp[i] = dto.OrderItemResponse{
			BookID:      it.BookID,
			Title:       it.Title,
			CoverImage:  it.CoverImage,
			Price:       it.Price,
			Quantity:    it.Quantity,
			WeightGrams: it.WeightGrams,
			Subtotal:    it.Subtotal,
		}
	}

	// Dynamic Internal Tracking URL
	internalTrackingURL := fmt.Sprintf("/orders/%s/tracking", order.InvoiceNumber)

	shippingStatus := "pending"
	if order.Shipment != nil {
		shippingStatus = string(order.Shipment.Status)
	}

	detail := dto.OrderDetailResponse{
		ID:                 order.ID,
		InvoiceNumber:      order.InvoiceNumber,
		Status:             string(order.Status),
		PaymentDeadline:    order.PaymentDeadline,
		PaidAt:             order.PaidAt,
		CancelledAt:        order.CancelledAt,
		CancellationReason: order.CancellationReason,
		ReturnedAt:         order.ReturnedAt,
		ReturnReason:       order.ReturnReason,
		CreatedAt:          order.CreatedAt,
		Recipient: dto.OrderRecipientResponse{
			Name:        order.RecipientName,
			Phone:       order.RecipientPhone,
			Address:     order.RecipientAddress,
			Province:    order.ProvinceName,
			City:        order.CityName,
			District:    order.DistrictName,
			Subdistrict: order.SubdistrictName,
			PostalCode:  order.PostalCode,
		},
		Shipping: dto.OrderShippingResponse{
			Courier:        order.Courier,
			CourierService: order.CourierService,
			TrackingNumber: order.ShippingTrackingNumber,
			Status:         shippingStatus,
			TrackingURL:    internalTrackingURL,
		},
		Pricing: dto.OrderPricingResponse{
			ItemsSubtotal:  order.ItemsSubtotal,
			ShippingCost:   order.ShippingCost,
			DiscountAmount: order.DiscountAmount,
			AdminFee:       order.AdminFee,
			TotalAmount:    order.TotalAmount,
		},
		Items: itemsResp,
	}

	if order.Payment != nil {
		bank := order.Payment.Bank
		billKey := order.Payment.BillKey
		paymentCode := order.Payment.PaymentCode
		store := order.Payment.Store

		if order.Payment.PaymentType == "cstore" {
			if paymentCode == nil {
				paymentCode = order.Payment.BillKey
			}
			if store == nil {
				store = order.Payment.Bank
			}
			bank = nil
			billKey = nil
		}

		detail.Payment = &dto.OrderPaymentResponse{
			PaymentType: order.Payment.PaymentType,
			Bank:        bank,
			VANumber:    order.Payment.VANumber,
			BillKey:     billKey,
			BillerCode:  order.Payment.BillerCode,
			PaymentCode: paymentCode,
			Store:       store,
			QRURL:       order.Payment.QRURL,
			DeeplinkURL: order.Payment.DeeplinkURL,
			Status:      string(order.Payment.Status),
			PaidAt:      order.Payment.PaidAt,
		}
	}

	return response.Success(c, fiber.StatusOK, "order detail retrieved successfully", detail)
}

func (h *OrderHandler) AdminFindAllOrders(c fiber.Ctx) error {
	pq := pagination.FromFiber(c)

	filter := repository.AdminOrderFilter{
		Status: strings.TrimSpace(c.Query("status")),
		Search: strings.TrimSpace(c.Query("search")),
		SortBy: strings.TrimSpace(c.Query("sort_by", "newest")),
	}

	if start := strings.TrimSpace(c.Query("start_date")); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			filter.StartDate = &utc
		} else {
			return response.Error(c, fiber.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD", nil)
		}
	}

	if end := strings.TrimSpace(c.Query("end_date")); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			utc := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
			filter.EndDate = &utc
		} else {
			return response.Error(c, fiber.StatusBadRequest, "invalid end_date format, expected YYYY-MM-DD", nil)
		}
	}

	items, meta, err := h.orderUsecase.AdminFindAllOrders(c.Context(), filter, pq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "orders retrieved successfully", items, meta)
}

func (h *OrderHandler) AdminGetOrderDetail(c fiber.Ctx) error {
	rawInvoice := c.Params("invoice_number")
	invoiceNumber, err := url.QueryUnescape(rawInvoice)
	if err != nil {
		invoiceNumber = rawInvoice
	}
	invoiceNumber = strings.TrimSpace(invoiceNumber)

	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	detail, err := h.orderUsecase.AdminGetOrderDetail(c.Context(), invoiceNumber)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, "order not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "order detail retrieved successfully", detail)
}

func (h *OrderHandler) DownloadInvoicePDF(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	rawInvoice := c.Params("invoice_number")
	invoiceNumber, err := url.QueryUnescape(rawInvoice)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid invoice number", nil)
	}

	isAdmin, _ := c.Locals("is_admin").(bool)

	pdfBytes, filename, err := h.orderUsecase.GetOrderInvoicePDF(c.Context(), userID, invoiceNumber, isAdmin)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, "order not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.Send(pdfBytes)
}
