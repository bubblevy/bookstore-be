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

type CartHandler struct {
	usecase  usecase.CartUsecase
	validate *validator.Validate
}

func NewCartHandler(r fiber.Router, uc usecase.CartUsecase, v *validator.Validate, authMiddleware fiber.Handler) {
	h := &CartHandler{
		usecase:  uc,
		validate: v,
	}

	cartGroup := r.Group("/cart", authMiddleware)
	cartGroup.Get("/", h.GetCart)
	cartGroup.Post("/items", h.AddItem)
	cartGroup.Post("/selected", h.GetSelectedItems)
	cartGroup.Patch("/items/:id", h.UpdateItem)
	cartGroup.Delete("/items/:id", h.DeleteItem)
	cartGroup.Delete("/", h.ClearCart)
}

func (h *CartHandler) getUserID(c fiber.Ctx) (uint, error) {
	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return 0, fiber.ErrUnauthorized
	}
	switch v := userIDVal.(type) {
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

func (h *CartHandler) GetCart(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	res, err := h.usecase.GetCart(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "cart retrieved successfully", res)
}

func (h *CartHandler) AddItem(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	var req dto.AddToCartRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.AddToCartInput{
		BookID:   req.BookID,
		Quantity: req.Quantity,
	}

	if err := h.usecase.AddToCart(c.Context(), userID, input); err != nil {
		switch {
		case errors.Is(err, entity.ErrBookNotActiveInCart):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrInsufficientStock):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusCreated, "item added to cart successfully", nil)
}

func (h *CartHandler) UpdateItem(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	idParam, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid item id", nil)
	}

	var req dto.UpdateCartItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.UpdateCartItemInput{
		Quantity: req.Quantity,
	}

	if err := h.usecase.UpdateItem(c.Context(), userID, uint(idParam), input); err != nil {
		switch {
		case errors.Is(err, entity.ErrCartItemNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrInsufficientStock):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "cart item updated successfully", nil)
}

func (h *CartHandler) DeleteItem(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	idParam, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid item id", nil)
	}

	if err := h.usecase.DeleteItem(c.Context(), userID, uint(idParam)); err != nil {
		if errors.Is(err, entity.ErrCartItemNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "cart item deleted successfully", nil)
}

func (h *CartHandler) ClearCart(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	if err := h.usecase.ClearCart(c.Context(), userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "cart cleared successfully", nil)
}

func (h *CartHandler) GetSelectedItems(c fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	var req dto.SelectedCartItemsRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	res, err := h.usecase.GetSelectedItems(c.Context(), userID, usecase.GetSelectedCartItemsInput{
		ItemIDs: req.ItemIDs,
	})
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "selected items calculated successfully", res)
}
