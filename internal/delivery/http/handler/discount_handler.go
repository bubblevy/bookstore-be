package handler

import (
	"errors"
	"strconv"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type DiscountHandler struct {
	usecase  usecase.DiscountUsecase
	validate *validator.Validate
}

func NewDiscountHandler(r fiber.Router, uc usecase.DiscountUsecase, v *validator.Validate, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &DiscountHandler{
		usecase:  uc,
		validate: v,
	}

	discounts := r.Group("/discounts")

	// Endpoint User & Checkout (Cek kupon)
	discounts.Post("/check", authMiddleware, h.CheckDiscount)

	// Admin Protected Endpoints
	adminRoutes := discounts.Group("", authMiddleware, adminMiddleware)
	adminRoutes.Get("/", h.FindAll)
	adminRoutes.Get("/:id", h.FindByID)
	adminRoutes.Post("/", h.Create)
	adminRoutes.Put("/:id", h.Update)
	adminRoutes.Delete("/:id", h.Delete)
}

func (h *DiscountHandler) FindAll(c fiber.Ctx) error {
	pq := pagination.FromFiber(c)
	filter := repository.DiscountFilter{
		Search: c.Query("search"),
	}

	if active := c.Query("is_active"); active != "" {
		val := active == "true" || active == "1"
		filter.IsActive = &val
	}

	if valid := c.Query("only_valid"); valid == "true" || valid == "1" {
		filter.OnlyValid = true
	}

	items, meta, err := h.usecase.FindAll(c.Context(), filter, pq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "discounts retrieved successfully", items, meta)
}

func (h *DiscountHandler) FindByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid discount id parameter", nil)
	}

	res, err := h.usecase.FindByID(c.Context(), uint(id))
	if err != nil {
		if errors.Is(err, entity.ErrDiscountNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "discount retrieved successfully", res)
}

func (h *DiscountHandler) Create(c fiber.Ctx) error {
	var req dto.CreateDiscountRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	tiers := make([]usecase.DiscountTierInput, 0, len(req.Tiers))
	for _, t := range req.Tiers {
		tiers = append(tiers, usecase.DiscountTierInput{
			MinSpend:    t.MinSpend,
			RewardType:  t.RewardType,
			RewardValue: t.RewardValue,
			MaxReward:   t.MaxReward,
		})
	}

	input := usecase.CreateDiscountInput{
		Code:              req.Code,
		Name:              req.Name,
		Description:       req.Description,
		DiscountType:      req.DiscountType,
		DiscountValue:     req.DiscountValue,
		MinPurchase:       req.MinPurchase,
		MaxDiscountAmount: req.MaxDiscountAmount,
		Quota:             req.Quota,
		IsAllBooks:        *req.IsAllBooks,
		IsActive:          req.IsActive,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		BookIDs:           req.BookIDs,
		Tiers:             tiers,
	}

	res, err := h.usecase.Create(c.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrDiscountCodeConflict):
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		case errors.Is(err, entity.ErrDiscountBookNotFound):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrDiscountTierRequired):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusCreated, "discount created successfully", res)
}

func (h *DiscountHandler) Update(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid discount id parameter", nil)
	}

	var req dto.UpdateDiscountRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	tiers := make([]usecase.DiscountTierInput, 0, len(req.Tiers))
	for _, t := range req.Tiers {
		tiers = append(tiers, usecase.DiscountTierInput{
			MinSpend:    t.MinSpend,
			RewardType:  t.RewardType,
			RewardValue: t.RewardValue,
			MaxReward:   t.MaxReward,
		})
	}

	input := usecase.UpdateDiscountInput{
		Code:              req.Code,
		Name:              req.Name,
		Description:       req.Description,
		DiscountType:      req.DiscountType,
		DiscountValue:     req.DiscountValue,
		MinPurchase:       req.MinPurchase,
		MaxDiscountAmount: req.MaxDiscountAmount,
		Quota:             req.Quota,
		IsAllBooks:        *req.IsAllBooks,
		IsActive:          req.IsActive,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		BookIDs:           req.BookIDs,
		Tiers:             tiers,
	}

	res, err := h.usecase.Update(c.Context(), uint(id), input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrDiscountNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrDiscountCodeConflict):
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		case errors.Is(err, entity.ErrDiscountBookNotFound):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrDiscountTierRequired):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "discount updated successfully", res)
}

func (h *DiscountHandler) Delete(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid discount id parameter", nil)
	}

	if err := h.usecase.Delete(c.Context(), uint(id)); err != nil {
		if errors.Is(err, entity.ErrDiscountNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "discount deleted successfully", nil)
}

func (h *DiscountHandler) CheckDiscount(c fiber.Ctx) error {
	var req dto.CheckDiscountRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	items := make([]usecase.CartItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, usecase.CartItemInput{
			BookID:   it.BookID,
			Quantity: it.Quantity,
		})
	}

	res, err := h.usecase.ValidateAndCalculate(c.Context(), req.Code, items)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrDiscountNotFound):
			return response.Error(c, fiber.StatusNotFound, "voucher code not found", nil)
		case errors.Is(err, entity.ErrDiscountInactive),
			errors.Is(err, entity.ErrDiscountNotStarted),
			errors.Is(err, entity.ErrDiscountExpired),
			errors.Is(err, entity.ErrDiscountQuotaExhaust),
			errors.Is(err, entity.ErrDiscountMinPurchase),
			errors.Is(err, entity.ErrDiscountNotApplicable):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "voucher applied successfully", res)
}
