package handler

import (
	"errors"
	"strconv"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type AddressHandler struct {
	usecase  usecase.AddressUsecase
	validate *validator.Validate
}

func NewAddressHandler(r fiber.Router, uc usecase.AddressUsecase, v *validator.Validate, authMiddleware fiber.Handler) {
	h := &AddressHandler{
		usecase:  uc,
		validate: v,
	}

	addr := r.Group("/addresses", authMiddleware)
	addr.Post("/", h.Create)
	addr.Get("/", h.GetAll)
	addr.Get("/primary", h.GetPrimary)
	addr.Get("/:id", h.GetByID)
	addr.Put("/:id", h.Update)
	addr.Delete("/:id", h.Delete)
	addr.Patch("/:id/primary", h.SetPrimary)
}

func (h *AddressHandler) Create(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	var req dto.CreateAddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.CreateAddressInput{
		ReceiverName:    req.ReceiverName,
		Phone:           req.Phone,
		AddressLine:     req.AddressLine,
		BiteshipAreaID:  req.BiteshipAreaID,
		ProvinceName:    req.ProvinceName,
		CityName:        req.CityName,
		DistrictName:    req.DistrictName,
		SubdistrictName: req.SubdistrictName,
		PostalCode:      req.PostalCode,
		IsPrimary:       req.IsPrimary,
	}

	res, err := h.usecase.CreateAddress(c.Context(), userID, input)
	if err != nil {
		if errors.Is(err, entity.ErrAddressLimitReached) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "address created successfully", res)
}

func (h *AddressHandler) GetAll(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	results, err := h.usecase.GetUserAddresses(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "addresses retrieved successfully", results)
}

func (h *AddressHandler) GetByID(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id", nil)
	}

	res, err := h.usecase.GetAddressByID(c.Context(), userID, uint(id))
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrAddressNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrUnauthorizedAddress):
			return response.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "address retrieved successfully", res)
}

func (h *AddressHandler) GetPrimary(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	res, err := h.usecase.GetPrimaryAddress(c.Context(), userID)
	if err != nil {
		if errors.Is(err, entity.ErrAddressNotFound) {
			return response.Error(c, fiber.StatusNotFound, "primary address not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "primary address retrieved successfully", res)
}

func (h *AddressHandler) Update(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id", nil)
	}

	var req dto.UpdateAddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.UpdateAddressInput{
		ReceiverName:    req.ReceiverName,
		Phone:           req.Phone,
		AddressLine:     req.AddressLine,
		BiteshipAreaID:  req.BiteshipAreaID,
		ProvinceName:    req.ProvinceName,
		CityName:        req.CityName,
		DistrictName:    req.DistrictName,
		SubdistrictName: req.SubdistrictName,
		PostalCode:      req.PostalCode,
		IsPrimary:       req.IsPrimary,
	}

	res, err := h.usecase.UpdateAddress(c.Context(), userID, uint(id), input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrAddressNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrUnauthorizedAddress):
			return response.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "address updated successfully", res)
}

func (h *AddressHandler) Delete(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id", nil)
	}

	err = h.usecase.DeleteAddress(c.Context(), userID, uint(id))
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrAddressNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrUnauthorizedAddress):
			return response.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case errors.Is(err, entity.ErrPrimaryAddressDelete):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "address deleted successfully", nil)
}

func (h *AddressHandler) SetPrimary(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid address id", nil)
	}

	err = h.usecase.SetPrimaryAddress(c.Context(), userID, uint(id))
	if err != nil {
		if errors.Is(err, entity.ErrAddressNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "primary address updated successfully", nil)
}
