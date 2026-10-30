package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type CheckoutHandler struct {
	usecase  usecase.CheckoutUsecase
	validate *validator.Validate
}

func NewCheckoutHandler(r fiber.Router, uc usecase.CheckoutUsecase, v *validator.Validate, authMiddleware fiber.Handler) {
	h := &CheckoutHandler{
		usecase:  uc,
		validate: v,
	}

	checkout := r.Group("/checkout", authMiddleware)
	checkout.Post("/preview", h.PreviewCheckout)
	checkout.Post("/order", h.CreateOrder)
	checkout.Get("/payment-methods", h.GetPaymentMethods)

	r.Post("/payments/midtrans/webhook", h.HandleMidtransWebhook)
}

func (h *CheckoutHandler) PreviewCheckout(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	var req dto.CheckoutPreviewRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
		}
	} else {
		_ = c.Bind().Query(&req)
	}
	if len(req.CartItemIDs) == 0 {
		idsStr := c.Query("cart_item_ids")
		if idsStr != "" {
			for _, part := range strings.Split(idsStr, ",") {
				part = strings.TrimSpace(part)
				if id, err := strconv.ParseUint(part, 10, 32); err == nil && id > 0 {
					req.CartItemIDs = append(req.CartItemIDs, uint(id))
				}
			}
		}
	}
	if req.AddressID == nil {
		if addrStr := c.Query("address_id"); addrStr != "" {
			if id, err := strconv.ParseUint(addrStr, 10, 32); err == nil {
				uid := uint(id)
				req.AddressID = &uid
			}
		}
	}
	if req.Courier == "" {
		req.Courier = c.Query("courier")
	}
	if req.CourierService == "" {
		req.CourierService = c.Query("courier_service")
	}
	if req.DiscountCode == nil {
		if dc := c.Query("discount_code"); dc != "" {
			req.DiscountCode = &dc
		}
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = c.Query("payment_method")
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.PreviewCheckoutInput{
		CartItemIDs:    req.CartItemIDs,
		AddressID:      req.AddressID,
		Courier:        req.Courier,
		CourierService: req.CourierService,
		DiscountCode:   req.DiscountCode,
		PaymentMethod:  req.PaymentMethod,
	}

	res, err := h.usecase.PreviewCheckout(c.Context(), userID, input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrCartEmpty),
			errors.Is(err, entity.ErrInsufficientStock),
			errors.Is(err, entity.ErrDiscountNotFound),
			errors.Is(err, entity.ErrDiscountInactive),
			errors.Is(err, entity.ErrDiscountNotStarted),
			errors.Is(err, entity.ErrDiscountExpired),
			errors.Is(err, entity.ErrDiscountQuotaExhaust),
			errors.Is(err, entity.ErrDiscountMinPurchase),
			errors.Is(err, entity.ErrDiscountNotApplicable),
			errors.Is(err, entity.ErrDiscountBookNotFound),
			errors.Is(err, entity.ErrInvalidPaymentMethod),
			errors.Is(err, entity.ErrPaymentMethodInactive),
			errors.Is(err, entity.ErrInvalidCourierService):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrAddressNotFound), errors.Is(err, entity.ErrCartItemNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
	}

	itemsResp := make([]dto.CheckoutItemPreview, len(res.Items))
	for i, it := range res.Items {
		itemsResp[i] = dto.CheckoutItemPreview{
			BookID:      it.BookID,
			Title:       it.Title,
			CoverImage:  it.CoverImage,
			Price:       it.Price,
			Quantity:    it.Quantity,
			WeightGrams: it.WeightGrams,
			Subtotal:    it.Subtotal,
		}
	}

	respData := dto.CheckoutPreviewResponse{
		Items:                   itemsResp,
		ItemsSubtotal:           res.ItemsSubtotal,
		TotalWeightGrams:        res.TotalWeightGrams,
		Courier:                 res.Courier,
		CourierService:          res.CourierService,
		ShippingCost:            res.ShippingCost,
		DiscountCode:            res.DiscountCode,
		DiscountAmount:          res.DiscountAmount,
		PaymentMethod:           res.PaymentMethod,
		AdminFee:                res.AdminFee,
		TotalPayment:            res.TotalPayment,
		AvailableCouriers:       res.AvailableCouriers,
		AvailablePaymentMethods: res.AvailablePaymentMethods,
	}

	if res.Address != nil {
		respData.Address = &dto.AddressPreviewResponse{
			ID:              res.Address.ID,
			ReceiverName:    res.Address.ReceiverName,
			Phone:           res.Address.Phone,
			AddressLine:     res.Address.AddressLine,
			ProvinceName:    res.Address.ProvinceName,
			CityName:        res.Address.CityName,
			DistrictName:    res.Address.DistrictName,
			SubdistrictName: res.Address.SubdistrictName,
			PostalCode:      res.Address.PostalCode,
		}
	}

	return response.Success(c, fiber.StatusOK, "checkout preview calculated successfully", respData)
}

func (h *CheckoutHandler) CreateOrder(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	var req dto.CreateOrderRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.CreateOrderInput{
		CartItemIDs:    req.CartItemIDs,
		AddressID:      req.AddressID,
		Courier:        req.Courier,
		CourierService: req.CourierService,
		DiscountCode:   req.DiscountCode,
		PaymentMethod:  req.PaymentMethod,
	}

	res, err := h.usecase.CreateOrder(c.Context(), userID, input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrCartEmpty),
			errors.Is(err, entity.ErrInsufficientStock),
			errors.Is(err, entity.ErrDiscountNotFound),
			errors.Is(err, entity.ErrDiscountExpired),
			errors.Is(err, entity.ErrDiscountQuotaExhaust),
			errors.Is(err, entity.ErrDiscountMinPurchase),
			errors.Is(err, entity.ErrDiscountNotApplicable),
			errors.Is(err, entity.ErrInvalidPaymentMethod),
			errors.Is(err, entity.ErrPaymentMethodInactive),
			errors.Is(err, entity.ErrInvalidCourierService):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrAddressNotFound), errors.Is(err, entity.ErrCartItemNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrMidtransChargeFailed):
			return response.Error(c, fiber.StatusBadGateway, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
	}

	respData := dto.CreateOrderResponse{
		InvoiceNumber:   res.InvoiceNumber,
		Status:          res.Status,
		TotalAmount:     res.TotalAmount,
		PaymentDeadline: res.PaymentDeadline,
		PaymentInfo: dto.PaymentInfoResponse{
			PaymentMethod: res.PaymentInfo.PaymentMethod,
			PaymentType:   res.PaymentInfo.PaymentType,
			Bank:        res.PaymentInfo.Bank,
			VANumber:    res.PaymentInfo.VANumber,
			BillKey:     res.PaymentInfo.BillKey,
			BillerCode:  res.PaymentInfo.BillerCode,
			PaymentCode: res.PaymentInfo.PaymentCode,
			Store:       res.PaymentInfo.Store,
			QRURL:         res.PaymentInfo.QRURL,
			DeeplinkURL:   res.PaymentInfo.DeeplinkURL,
			Instructions:  res.PaymentInfo.Instructions,
		},
	}

	return response.Success(c, fiber.StatusCreated, "order created successfully", respData)
}

func (h *CheckoutHandler) HandleMidtransWebhook(c fiber.Ctx) error {
	rawBody := append([]byte(nil), c.Body()...)
	var payload dto.MidtransWebhookPayload
	if err := c.Bind().Body(&payload); err != nil {
		if jsonErr := json.Unmarshal(rawBody, &payload); jsonErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "cannot parse notification body",
			})
		}
	}

	if len(rawBody) == 0 {
		rawBody, _ = json.Marshal(payload)
	}

	input := usecase.MidtransWebhookInput{
		TransactionTime:   payload.TransactionTime,
		TransactionStatus: payload.TransactionStatus,
		TransactionID:     payload.TransactionID,
		StatusMessage:     payload.StatusMessage,
		StatusCode:        payload.StatusCode,
		SignatureKey:      payload.SignatureKey,
		PaymentType:       payload.PaymentType,
		OrderID:           payload.OrderID,
		GrossAmount:       payload.GrossAmount,
		FraudStatus:       payload.FraudStatus,
		Currency:          payload.Currency,
		RawJSON:           rawBody,
	}

	if err := h.usecase.HandleMidtransWebhook(c.Context(), input); err != nil {
		if err.Error() == "invalid midtrans webhook signature" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "ok",
		"message": "notification handled successfully",
	})
}

func (h *CheckoutHandler) GetPaymentMethods(c fiber.Ctx) error {
	amount, _ := strconv.Atoi(c.Query("amount", "0"))
	res, err := h.usecase.GetPaymentMethods(c.Context(), amount)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "payment methods retrieved successfully", res)
}

