package handler

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type OrderHandler struct {
	orderUsecase usecase.OrderUsecase
}

func NewOrderHandler(r fiber.Router, orderUC usecase.OrderUsecase, authMiddleware fiber.Handler) {
	h := &OrderHandler{orderUsecase: orderUC}

	orders := r.Group("/orders", authMiddleware)
	orders.Get("/", h.GetUserOrders)
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
		detail.Payment = &dto.OrderPaymentResponse{
			PaymentType: order.Payment.PaymentType,
			Bank:        order.Payment.Bank,
			VANumber:    order.Payment.VANumber,
			BillKey:     order.Payment.BillKey,
			BillerCode:  order.Payment.BillerCode,
			QRURL:       order.Payment.QRURL,
			Status:      string(order.Payment.Status),
			PaidAt:      order.Payment.PaidAt,
		}
	}

	return response.Success(c, fiber.StatusOK, "order detail retrieved successfully", detail)
}
