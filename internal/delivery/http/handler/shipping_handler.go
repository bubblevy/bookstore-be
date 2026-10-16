package handler

import (
	"errors"
	"net/url"
	"strings"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type ShippingHandler struct {
	shippingUsecase usecase.ShippingUsecase
}

func NewShippingHandler(r fiber.Router, u usecase.ShippingUsecase, authMiddleware fiber.Handler, adminMiddleware fiber.Handler, webhookAuthMiddleware fiber.Handler) {
	h := &ShippingHandler{shippingUsecase: u}

	// Public Routes
	r.Get("/shipping/areas", h.SearchAreas)
	r.Get("/shipping/couriers", h.GetCouriers)

	// Webhook Route
	r.Post("/shipping/webhook", webhookAuthMiddleware, h.HandleWebhook)

	// Customer & Admin Tracking Route
	r.Get("/orders/:invoice_number/tracking", authMiddleware, h.TrackOrder)

	// Admin Order Fulfillment Route
	r.Post("/orders/:invoice_number/fulfill", authMiddleware, adminMiddleware, h.FulfillOrder)
	r.Post("/orders/:invoice_number/return", authMiddleware, adminMiddleware, h.ProcessReturn)
}

func (h *ShippingHandler) SearchAreas(c fiber.Ctx) error {
	keyword := strings.TrimSpace(c.Query("search"))
	if len(keyword) < 3 {
		return response.Error(c, fiber.StatusBadRequest, "search parameter must be at least 3 characters", nil)
	}

	areas, err := h.shippingUsecase.SearchAreas(c.Context(), keyword)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "areas retrieved successfully", areas)
}

func (h *ShippingHandler) GetCouriers(c fiber.Ctx) error {
	couriers, err := h.shippingUsecase.GetCouriers(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "couriers retrieved successfully", couriers)
}

func (h *ShippingHandler) FulfillOrder(c fiber.Ctx) error {
	rawInvoice := c.Params("invoice_number")
	invoiceNumber, err := url.QueryUnescape(rawInvoice)
	if err != nil {
		invoiceNumber = rawInvoice
	}
	invoiceNumber = strings.TrimSpace(invoiceNumber)

	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	var req struct {
		OrderNote string `json:"order_note"`
	}
	_ = c.Bind().Body(&req)

	result, err := h.shippingUsecase.FulfillOrder(c.Context(), invoiceNumber, req.OrderNote)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "pickup scheduled successfully", result)
}

func (h *ShippingHandler) TrackOrder(c fiber.Ctx) error {
	invoiceNumber := strings.TrimSpace(c.Params("invoice_number"))
	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	userID, _ := c.Locals("user_id").(uint)

	result, err := h.shippingUsecase.GetOrderTracking(c.Context(), userID, invoiceNumber)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) || errors.Is(err, entity.ErrShipmentNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, entity.ErrUnauthorizedTracking) {
			return response.Error(c, fiber.StatusForbidden, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "tracking details retrieved successfully", result)
}

func (h *ShippingHandler) ProcessReturn(c fiber.Ctx) error {
	invoiceNumber := strings.TrimSpace(c.Params("invoice_number"))
	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind().Body(&req)

	if err := h.shippingUsecase.ProcessReturn(c.Context(), invoiceNumber, req.Reason); err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "order return processed and stock restored successfully", nil)
}

func (h *ShippingHandler) HandleWebhook(c fiber.Ctx) error {
	var payload biteship.WebhookTrackingPayload
	if err := c.Bind().Body(&payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid webhook payload", nil)
	}

	if err := h.shippingUsecase.HandleWebhook(c.Context(), payload); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "webhook processed successfully", nil)
}
