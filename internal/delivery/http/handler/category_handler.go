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

type CategoryHandler struct {
	usecase  usecase.CategoryUsecase
	validate *validator.Validate
}

func NewCategoryHandler(r fiber.Router, uc usecase.CategoryUsecase, v *validator.Validate, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &CategoryHandler{
		usecase:  uc,
		validate: v,
	}

	categories := r.Group("/categories")

	// Public routes
	categories.Get("/", h.FindAll)
	categories.Get("/:slug", h.FindBySlug)

	// Protected routes (Admin)
	adminRoutes := categories.Group("", authMiddleware, adminMiddleware)
	adminRoutes.Post("/", h.Create)
	adminRoutes.Put("/:id", h.Update)
	adminRoutes.Delete("/:id", h.Delete)
}

func (h *CategoryHandler) Create(c fiber.Ctx) error {
	var req dto.CreateCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.CreateCategoryInput{
		Name:     req.Name,
		ImageURL: req.ImageURL,
	}

	res, err := h.usecase.Create(c.Context(), input)
	if err != nil {
		if errors.Is(err, entity.ErrCategorySlugConflict) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "category created successfully", res)
}

func (h *CategoryHandler) FindAll(c fiber.Ctx) error {
	res, err := h.usecase.FindAll(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "categories retrieved successfully", res)
}

func (h *CategoryHandler) FindBySlug(c fiber.Ctx) error {
	slugStr := c.Params("slug")
	res, err := h.usecase.FindBySlug(c.Context(), slugStr)
	if err != nil {
		if errors.Is(err, entity.ErrCategoryNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "category retrieved successfully", res)
}

func (h *CategoryHandler) Update(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid ID parameter", nil)
	}

	var req dto.UpdateCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	input := usecase.UpdateCategoryInput{
		ID:       uint(id),
		Name:     req.Name,
		ImageURL: req.ImageURL,
	}

	res, err := h.usecase.Update(c.Context(), uint(id), input)
	if err != nil {
		if errors.Is(err, entity.ErrCategoryNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, entity.ErrCategorySlugConflict) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "category updated successfully", res)
}

func (h *CategoryHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid ID parameter", nil)
	}

	if err := h.usecase.Delete(c.Context(), uint(id)); err != nil {
		if errors.Is(err, entity.ErrCategoryNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "category deleted successfully", nil)
}
