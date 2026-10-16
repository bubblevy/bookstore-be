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

type BookHandler struct {
	usecase  usecase.BookUsecase
	validate *validator.Validate
}

func NewBookHandler(r fiber.Router, uc usecase.BookUsecase, v *validator.Validate, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &BookHandler{
		usecase:  uc,
		validate: v,
	}

	books := r.Group("/books")

	// Public Endpoints
	books.Get("/", h.FindAll)
	books.Get("/:slug", h.FindBySlug)

	// Admin Protected Endpoints
	adminRoutes := books.Group("", authMiddleware, adminMiddleware)
	adminRoutes.Post("/", h.Create)
	adminRoutes.Put("/:id", h.Update)
	adminRoutes.Delete("/:id", h.Delete)
}

func (h *BookHandler) FindAll(c fiber.Ctx) error {
	pq := pagination.FromFiber(c)
	filter := repository.BookFilter{
		Search:        c.Query("search"),
		CategorySlug:  c.Query("category"),
		AuthorSlug:    c.Query("author"),
		PublisherSlug: c.Query("publisher"),
		SortBy:        c.Query("sort_by", "newest"),
	}

	if minP := c.Query("min_price"); minP != "" {
		if val, err := strconv.ParseFloat(minP, 64); err == nil && val >= 0 {
			filter.MinPrice = &val
		} else {
			return response.Error(c, fiber.StatusBadRequest, "invalid min_price parameter", nil)
		}
	}

	if maxP := c.Query("max_price"); maxP != "" {
		if val, err := strconv.ParseFloat(maxP, 64); err == nil && val >= 0 {
			filter.MaxPrice = &val
		} else {
			return response.Error(c, fiber.StatusBadRequest, "invalid max_price parameter", nil)
		}
	}

	// Security Policy: Regular users can ONLY view active books
	isAdmin, _ := c.Locals("is_admin").(bool)
	if !isAdmin {
		activeTrue := true
		filter.IsActive = &activeTrue
	} else {
		if active := c.Query("is_active"); active != "" {
			if active == "all" {
				filter.IsActive = nil // Return both active and inactive
			} else {
				val := active == "true" || active == "1"
				filter.IsActive = &val
			}
		}
	}

	items, meta, err := h.usecase.FindAll(c.Context(), filter, pq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "books retrieved successfully", items, meta)
}

func (h *BookHandler) FindBySlug(c fiber.Ctx) error {
	slugStr := c.Params("slug")
	res, err := h.usecase.FindBySlug(c.Context(), slugStr)
	if err != nil {
		if errors.Is(err, entity.ErrBookNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	// Security Check: If inactive, non-admins cannot view it directly
	isAdmin, _ := c.Locals("is_admin").(bool)
	if !res.IsActive && !isAdmin {
		return response.Error(c, fiber.StatusNotFound, entity.ErrBookNotFound.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "book retrieved successfully", res)
}

func (h *BookHandler) Create(c fiber.Ctx) error {
	var req dto.CreateBookRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	input := usecase.CreateBookInput{
		CategoryID:      req.CategoryID,
		AuthorID:        req.AuthorID,
		PublisherID:     req.PublisherID,
		Title:           req.Title,
		ISBN:            req.ISBN,
		Synopsis:        req.Synopsis,
		Price:           req.Price,
		WeightGrams:     req.WeightGrams,
		Pages:           req.Pages,
		PublicationYear: req.PublicationYear,
		Stock:           req.Stock,
		CoverImage:      req.CoverImage,
		IsActive:        req.IsActive,
	}

	res, err := h.usecase.Create(c.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrBookCategoryNotFound),
			errors.Is(err, entity.ErrBookAuthorNotFound),
			errors.Is(err, entity.ErrBookPublisherNotFound):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrBookSlugConflict),
			errors.Is(err, entity.ErrBookISBNConflict):
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusCreated, "book created successfully", res)
}

func (h *BookHandler) Update(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid book id parameter", nil)
	}

	var req dto.UpdateBookRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	input := usecase.UpdateBookInput{
		CategoryID:      req.CategoryID,
		AuthorID:        req.AuthorID,
		PublisherID:     req.PublisherID,
		Title:           req.Title,
		ISBN:            req.ISBN,
		Synopsis:        req.Synopsis,
		Price:           req.Price,
		WeightGrams:     req.WeightGrams,
		Pages:           req.Pages,
		PublicationYear: req.PublicationYear,
		Stock:           req.Stock,
		CoverImage:      req.CoverImage,
		IsActive:        req.IsActive,
	}

	res, err := h.usecase.Update(c.Context(), uint(id), input)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrBookNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrBookCategoryNotFound),
			errors.Is(err, entity.ErrBookAuthorNotFound),
			errors.Is(err, entity.ErrBookPublisherNotFound):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrBookSlugConflict),
			errors.Is(err, entity.ErrBookISBNConflict):
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "book updated successfully", res)
}

func (h *BookHandler) Delete(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid book id parameter", nil)
	}

	if err := h.usecase.Delete(c.Context(), uint(id)); err != nil {
		if errors.Is(err, entity.ErrBookNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "book deleted successfully", nil)
}
