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

type PublisherHandler struct {
	usecase  usecase.PublisherUsecase
	validate *validator.Validate
}

func NewPublisherHandler(r fiber.Router, uc usecase.PublisherUsecase, v *validator.Validate, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &PublisherHandler{
		usecase:  uc,
		validate: v,
	}

	publishers := r.Group("/publishers")

	// Public routes
	publishers.Get("/", h.FindAll)
	publishers.Get("/:slug", h.FindBySlug)

	// Admin protected routes
	adminRoutes := publishers.Group("", authMiddleware, adminMiddleware)
	adminRoutes.Post("/", h.Create)
	adminRoutes.Put("/:id", h.Update)
	adminRoutes.Delete("/:id", h.Delete)
}

func (h *PublisherHandler) Create(c fiber.Ctx) error {
	var req dto.CreatePublisherRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	res, err := h.usecase.Create(c.Context(), usecase.CreatePublisherInput{Name: req.Name})
	if err != nil {
		if errors.Is(err, entity.ErrPublisherSlugConflict) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "publisher created successfully", res)
}

func (h *PublisherHandler) FindAll(c fiber.Ctx) error {
	res, err := h.usecase.FindAll(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "publishers retrieved successfully", res)
}

func (h *PublisherHandler) FindBySlug(c fiber.Ctx) error {
	slugStr := c.Params("slug")
	res, err := h.usecase.FindBySlug(c.Context(), slugStr)
	if err != nil {
		if errors.Is(err, entity.ErrPublisherNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}
	return response.Success(c, fiber.StatusOK, "publisher retrieved successfully", res)
}

func (h *PublisherHandler) Update(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid publisher id parameter", nil)
	}

	var req dto.UpdatePublisherRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	res, err := h.usecase.Update(c.Context(), uint(id), usecase.UpdatePublisherInput{Name: req.Name})
	if err != nil {
		if errors.Is(err, entity.ErrPublisherNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, entity.ErrPublisherSlugConflict) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "publisher updated successfully", res)
}

func (h *PublisherHandler) Delete(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid publisher id parameter", nil)
	}

	if err := h.usecase.Delete(c.Context(), uint(id)); err != nil {
		if errors.Is(err, entity.ErrPublisherNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "publisher deleted successfully", nil)
}
