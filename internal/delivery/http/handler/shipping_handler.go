package handler

import (
	"errors"
	"strconv"
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

func NewShippingHandler(r fiber.Router, u usecase.ShippingUsecase, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &ShippingHandler{shippingUsecase: u}

	// Public Routes
	r.Get("/shipping/areas", h.SearchAreas)
	r.Get("/shipping/couriers", h.GetCouriers)

	// Webhook Route
	r.Post("/shipping/webhook", h.HandleWebhook)

	// User & Admin Tracking Route
	r.Get("/orders/:order_id/tracking", authMiddleware, h.TrackOrder)

	// Admin Dispatch Order Route
	admin := r.Group("/admin/orders", authMiddleware, adminMiddleware)
	admin.Post("/:order_id/fulfill", h.FulfillOrder)
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
	idParam, err := strconv.ParseUint(c.Params("order_id"), 10, 32)
	if err != nil || idParam == 0 {
		return response.Error(c, fiber.StatusBadRequest, "invalid order id", nil)
	}
	orderID := uint(idParam)

	var req struct {
		OrderNote string `json:"order_note"`
	}
	_ = c.Bind().Body(&req)

	result, err := h.shippingUsecase.FulfillOrder(c.Context(), orderID, req.OrderNote)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "pickup scheduled successfully", result)
}

func (h *ShippingHandler) TrackOrder(c fiber.Ctx) error {
	idParam, err := strconv.ParseUint(c.Params("order_id"), 10, 32)
	if err != nil || idParam == 0 {
		return response.Error(c, fiber.StatusBadRequest, "invalid order id", nil)
	}
	orderID := uint(idParam)

	userID, _ := c.Locals("user_id").(uint)

	result, err := h.shippingUsecase.GetOrderTracking(c.Context(), userID, orderID)
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
