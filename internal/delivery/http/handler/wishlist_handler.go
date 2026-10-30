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

type WishlistHandler struct {
	usecase  usecase.WishlistUsecase
	validate *validator.Validate
}

func NewWishlistHandler(r fiber.Router, uc usecase.WishlistUsecase, v *validator.Validate, authMiddleware fiber.Handler) {
	h := &WishlistHandler{
		usecase:  uc,
		validate: v,
	}

	wishlistGroup := r.Group("/wishlist", authMiddleware)
	wishlistGroup.Get("/", h.GetWishlist)
	wishlistGroup.Get("/check/:book_id", h.CheckWishlist)
	wishlistGroup.Post("/items", h.AddItem)
	wishlistGroup.Delete("/items/:book_id", h.DeleteItem)
	wishlistGroup.Post("/items/:book_id/move-to-cart", h.MoveToCart)
	wishlistGroup.Delete("/", h.ClearWishlist)
}

func (h *WishlistHandler) GetWishlist(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	pq := pagination.FromFiber(c)
	items, meta, err := h.usecase.GetWishlist(c.Context(), userID, pq.Page, pq.PerPage)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	resItems := make([]dto.WishlistItemResponse, 0, len(items))
	for _, it := range items {
		resItems = append(resItems, dto.WishlistItemResponse{
			ID:     it.ID,
			BookID: it.BookID,
			Book: dto.WishlistBookSummary{
				ID:                 it.Book.ID,
				Title:              it.Book.Title,
				Slug:               it.Book.Slug,
				Price:              it.Book.Price,
				OriginalPrice:      it.Book.OriginalPrice,
				DiscountPercentage: it.Book.DiscountPercentage,
				WeightGrams:        it.Book.WeightGrams,
				Stock:              it.Book.Stock,
				CoverImage:         it.Book.CoverImage,
				IsActive:           it.Book.IsActive,
				CategoryName:       it.Book.CategoryName,
				AuthorName:         it.Book.AuthorName,
				PublisherName:      it.Book.PublisherName,
			},
			CreatedAt: it.CreatedAt,
		})
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "wishlist retrieved successfully", resItems, meta)
}

func (h *WishlistHandler) CheckWishlist(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	bookIDParam, err := strconv.ParseUint(c.Params("book_id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid book id", nil)
	}

	isWishlisted, err := h.usecase.CheckWishlist(c.Context(), userID, uint(bookIDParam))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "wishlist status retrieved", dto.WishlistCheckResponse{
		BookID:       uint(bookIDParam),
		IsWishlisted: isWishlisted,
	})
}

func (h *WishlistHandler) AddItem(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	var req dto.AddWishlistRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	if err := h.usecase.AddToWishlist(c.Context(), userID, req.BookID); err != nil {
		switch {
		case errors.Is(err, entity.ErrBookNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrBookNotActiveInWishlist):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, entity.ErrWishlistAlreadyExists):
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusCreated, "book added to wishlist successfully", nil)
}

func (h *WishlistHandler) DeleteItem(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	bookIDParam, err := strconv.ParseUint(c.Params("book_id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid book id", nil)
	}

	if err := h.usecase.RemoveFromWishlist(c.Context(), userID, uint(bookIDParam)); err != nil {
		if errors.Is(err, entity.ErrWishlistItemNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "book removed from wishlist successfully", nil)
}

func (h *WishlistHandler) MoveToCart(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	bookIDParam, err := strconv.ParseUint(c.Params("book_id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid book id", nil)
	}

	var req dto.MoveToCartRequest
	_ = c.Bind().Body(&req) // optional body

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	if err := h.usecase.MoveToCart(c.Context(), userID, uint(bookIDParam), quantity); err != nil {
		switch {
		case errors.Is(err, entity.ErrWishlistItemNotFound), errors.Is(err, entity.ErrBookNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case errors.Is(err, entity.ErrBookNotActiveInWishlist), errors.Is(err, entity.ErrInsufficientStock):
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
		}
	}

	return response.Success(c, fiber.StatusOK, "book moved to cart successfully", nil)
}

func (h *WishlistHandler) ClearWishlist(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	if err := h.usecase.ClearWishlist(c.Context(), userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "wishlist cleared successfully", nil)
}
