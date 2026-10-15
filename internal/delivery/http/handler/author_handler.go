package handler

import (
	"errors"
	"strconv"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type AuthorHandler struct {
	usecase  usecase.AuthorUsecase
	validate *validator.Validate
}

func NewAuthorHandler(r fiber.Router, uc usecase.AuthorUsecase, v *validator.Validate, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &AuthorHandler{
		usecase:  uc,
		validate: v,
	}

	authors := r.Group("/authors")

	// Public endpoints
	authors.Get("/", h.FindAll)
	authors.Get("/:slug", h.FindBySlug)

	// Admin protected endpoints
	adminRoutes := authors.Group("", authMiddleware, adminMiddleware)
	adminRoutes.Post("/", h.Create)
	adminRoutes.Put("/:id", h.Update)
	adminRoutes.Delete("/:id", h.Delete)
}

func (h *AuthorHandler) Create(c fiber.Ctx) error {
	var req dto.CreateAuthorRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	input := usecase.CreateAuthorInput{
		Name: req.Name,
		Bio:  req.Bio,
	}

	res, err := h.usecase.Create(c.Context(), input)
	if err != nil {
		if errors.Is(err, entity.ErrAuthorSlugConflict) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "author created successfully", res)
}

func (h *AuthorHandler) FindAll(c fiber.Ctx) error {
	pq := pagination.FromFiber(c)

	items, meta, err := h.usecase.FindAll(c.Context(), pq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "authors retrieved successfully", items, meta)
}

func (h *AuthorHandler) FindBySlug(c fiber.Ctx) error {
	slugStr := c.Params("slug")
	bookPq := pagination.FromFiber(c)

	res, err := h.usecase.FindBySlug(c.Context(), slugStr, bookPq)
	if err != nil {
		if errors.Is(err, entity.ErrAuthorNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "author retrieved successfully", res)
}

func (h *AuthorHandler) Update(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid author id parameter", nil)
	}

	var req dto.UpdateAuthorRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	input := usecase.UpdateAuthorInput{
		Name: req.Name,
		Bio:  req.Bio,
	}

	res, err := h.usecase.Update(c.Context(), uint(id), input)
	if err != nil {
		if errors.Is(err, entity.ErrAuthorNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, entity.ErrAuthorSlugConflict) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "author updated successfully", res)
}

func (h *AuthorHandler) Delete(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid author id parameter", nil)
	}

	if err := h.usecase.Delete(c.Context(), uint(id)); err != nil {
		if errors.Is(err, entity.ErrAuthorNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "author deleted successfully", nil)
}
