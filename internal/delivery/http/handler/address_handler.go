package handler

import (
	"errors"
	"strconv"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type AddressHandler struct {
	u        usecase.AddressUsecase
	validate *validator.Validate
}

func NewAddressHandler(r fiber.Router, u usecase.AddressUsecase, validate *validator.Validate, authMiddleware fiber.Handler) {
	h := &AddressHandler{
		u:        u,
		validate: validate,
	}

	routes := r.Group("/addresses", authMiddleware)
	routes.Post("/", middleware.RateLimit(middleware.StandardRateLimit), h.Create)
	routes.Get("/", middleware.RateLimit(middleware.StandardRateLimit), h.FindAll)
	routes.Get("/:id", middleware.RateLimit(middleware.StandardRateLimit), h.FindByID)
	routes.Put("/:id", middleware.RateLimit(middleware.StandardRateLimit), h.Update)
	routes.Delete("/:id", middleware.RateLimit(middleware.StandardRateLimit), h.Delete)
	routes.Patch("/:id/primary", middleware.RateLimit(middleware.StandardRateLimit), h.SetPrimary)
}

func (h *AddressHandler) Create(c fiber.Ctx) error {
	var req dto.CreateAddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	input := usecase.CreateAddressInput{
		UserID:          userID,
		ReceiverName:    req.ReceiverName,
		Phone:           req.Phone,
		AddressLine:     req.AddressLine,
		DestinationID:   req.DestinationID,
		ProvinceName:    req.ProvinceName,
		CityName:        req.CityName,
		DistrictName:    req.DistrictName,
		SubdistrictName: req.SubdistrictName,
		PostalCode:      req.PostalCode,
		IsPrimary:       req.IsPrimary,
	}

	res, err := h.u.Create(c.Context(), input)
	if err != nil {
		if errors.Is(err, entity.ErrAddressLimitReached) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}

		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "address created successfully", res)
}

func (h *AddressHandler) FindAll(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	res, err := h.u.GetAllByUser(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "addresses retrieved successfully", res)
}

func (h *AddressHandler) FindByID(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id parameter", nil)
	}

	res, err := h.u.GetByID(c.Context(), uint(id), userID)
	if err != nil {
		if errors.Is(err, entity.ErrAddressNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "address retrieved successfully", res)
}

func (h *AddressHandler) Update(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id parameter", nil)
	}

	var req dto.UpdateAddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	input := usecase.UpdateAddressInput{
		ID:              uint(id),
		UserID:          userID,
		ReceiverName:    req.ReceiverName,
		Phone:           req.Phone,
		AddressLine:     req.AddressLine,
		DestinationID:   req.DestinationID,
		ProvinceName:    req.ProvinceName,
		CityName:        req.CityName,
		DistrictName:    req.DistrictName,
		SubdistrictName: req.SubdistrictName,
		PostalCode:      req.PostalCode,
		IsPrimary:       req.IsPrimary,
	}

	res, err := h.u.Update(c.Context(), input)
	if err != nil {
		if errors.Is(err, entity.ErrAddressNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "address updated successfully", res)
}

func (h *AddressHandler) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id parameter", nil)
	}

	if err := h.u.Delete(c.Context(), uint(id), userID); err != nil {
		if errors.Is(err, entity.ErrAddressNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}

		if errors.Is(err, entity.ErrCannotDeletePrimaryAddress) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "address deleted successfully", nil)
}

func (h *AddressHandler) SetPrimary(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id parameter", nil)
	}

	if err := h.u.SetPrimary(c.Context(), uint(id), userID); err != nil {
		if errors.Is(err, entity.ErrAddressNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "primary address updated successfully", nil)
}
