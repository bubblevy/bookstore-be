package handler

import (
	"errors"
	"net/url"
	"strings"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ShippingHandler struct {
	shippingUsecase usecase.ShippingUsecase
	validate        *validator.Validate
}

func NewShippingHandler(
	r fiber.Router,
	u usecase.ShippingUsecase,
	v *validator.Validate,
	authMiddleware fiber.Handler,
	adminMiddleware fiber.Handler,
	webhookAuthMiddleware fiber.Handler,
) {
	h := &ShippingHandler{
		shippingUsecase: u,
		validate:        v,
	}

	// Public routes
	r.Get("/shipping/areas", h.SearchAreas)
	r.Get("/shipping/couriers", h.GetCouriers)

	// Pre-checkout dynamic rates calculation
	r.Post("/shipping/rates", authMiddleware, h.GetShippingRates)

	// Webhook endpoint
	r.Post("/shipping/webhook", webhookAuthMiddleware, h.HandleWebhook)

	// User / Admin tracking route
	r.Get("/orders/:invoice_number/tracking", authMiddleware, h.TrackOrder)

	// Admin fulfillment and return management
	r.Post("/orders/:invoice_number/fulfill", authMiddleware, adminMiddleware, h.FulfillOrder)
	r.Post("/orders/:invoice_number/return", authMiddleware, adminMiddleware, h.ProcessReturn)
}

func (h *ShippingHandler) getUserID(c fiber.Ctx) (uint, error) {
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

func (h *ShippingHandler) GetShippingRates(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	var req dto.GetShippingRatesRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}
	if h.validate != nil {
		if err := h.validate.Struct(req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
		}
	}

	input := usecase.GetShippingRatesInput{
		CartItemIDs: req.CartItemIDs,
		AddressID:   req.AddressID,
	}

	res, err := h.shippingUsecase.GetShippingRates(c.Context(), userID, input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrCartEmpty):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrAddressNotFound), errors.Is(err, entity.ErrCartItemNotFound), errors.Is(err, entity.ErrBookNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrShippingRateFailed):
			return response.Error(c, fiber.StatusBadGateway, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
	}

	options := make([]dto.CourierOptionResponse, len(res.Options))
	for i, opt := range res.Options {
		options[i] = dto.CourierOptionResponse{
			CourierCode:        opt.CourierCode,
			CourierName:        opt.CourierName,
			CourierServiceName: opt.CourierServiceName,
			CourierServiceCode: opt.CourierServiceCode,
			Description:        opt.Description,
			Price:              opt.Price,
			ETD:                opt.ETD,
			Type:               opt.Type,
		}
	}

	resp := dto.ShippingRatesResponse{
		TotalWeightGrams: res.TotalWeightGrams,
		DestinationArea:  res.DestinationArea,
		Options:          options,
	}

	return response.Success(c, fiber.StatusOK, "shipping rates retrieved successfully", resp)
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

	var req dto.FulfillOrderRequest
	_ = c.Bind().Body(&req)

	result, err := h.shippingUsecase.FulfillOrder(c.Context(), invoiceNumber, req.OrderNote)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrOrderNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrOrderNotPaid), errors.Is(err, entity.ErrOrderAlreadyFulfilled):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrDispatchCourierFailed):
			return response.Error(c, fiber.StatusBadGateway, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
	}

	resp := dto.FulfillOrderResponse{
		ShipmentID:      result.ShipmentID,
		InvoiceNumber:   result.InvoiceNumber,
		BiteshipOrderID: result.BiteshipOrderID,
		WaybillNumber:   result.WaybillNumber,
		Status:          string(result.Status),
		Courier:         result.Courier,
		CourierService:  result.CourierService,
		TrackingURL:     result.TrackingURL,
	}

	return response.Success(c, fiber.StatusOK, "pickup scheduled successfully", resp)
}

func (h *ShippingHandler) TrackOrder(c fiber.Ctx) error {
	invoiceNumber := strings.TrimSpace(c.Params("invoice_number"))
	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	result, err := h.shippingUsecase.GetOrderTracking(c.Context(), userID, invoiceNumber)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrOrderNotFound), errors.Is(err, entity.ErrShipmentNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrUnauthorizedTracking):
			return response.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case errors.Is(err, entity.ErrTrackingNumberNotReady):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "tracking details retrieved successfully", result)
}

func (h *ShippingHandler) ProcessReturn(c fiber.Ctx) error {
	invoiceNumber := strings.TrimSpace(c.Params("invoice_number"))
	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	var req dto.ProcessReturnRequest
	_ = c.Bind().Body(&req)

	if err := h.shippingUsecase.ProcessReturn(c.Context(), invoiceNumber, req.Reason); err != nil {
		switch {
		case errors.Is(err, entity.ErrOrderNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrOrderInvalidReturnState), errors.Is(err, entity.ErrOrderAlreadyReturned):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
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
