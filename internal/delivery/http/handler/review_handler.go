package handler

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ReviewHandler struct {
	usecase  usecase.ReviewUsecase
	validate *validator.Validate
}

func NewReviewHandler(
	r fiber.Router,
	uc usecase.ReviewUsecase,
	v *validator.Validate,
	authMiddleware fiber.Handler,
	adminMiddleware fiber.Handler,
) {
	h := &ReviewHandler{
		usecase:  uc,
		validate: v,
	}

	// public endpoint
	r.Get("/books/:slug/reviews", h.GetBookReviews)

	// customer review
	orders := r.Group("/orders", authMiddleware)
	orders.Get("/:invoice_number/review-eligibility", h.CheckOrderEligibility)
	orders.Post("/:invoice_number/reviews", h.SubmitOrderReviews)

	reviews := r.Group("/reviews", authMiddleware)
	reviews.Get("/my", h.GetMyReviews)
	reviews.Put("/:id", h.UpdateReview)

	// admin review management
	admin := r.Group("/admin/reviews", authMiddleware, adminMiddleware)
	admin.Get("/", h.AdminFindAllReviews)
	admin.Patch("/:id/visibility", h.AdminModerateReview)
	admin.Post("/:id/reply", h.AdminReplyReview)
	admin.Delete("/:id", h.AdminDeleteReview)
}

func (h *ReviewHandler) GetBookReviews(c fiber.Ctx) error {
	slug := strings.TrimSpace(c.Params("slug"))
	if slug == "" {
		return response.Error(c, fiber.StatusBadRequest, "book slug is required", nil)
	}

	pq := pagination.FromFiber(c)

	filter := repository.ReviewFilter{
		SortBy: strings.ToLower(strings.TrimSpace(c.Query("sort_by", "newest"))),
	}

	if ratingStr := c.Query("rating"); ratingStr != "" {
		if rVal, err := strconv.Atoi(ratingStr); err == nil && rVal >= 1 && rVal <= 5 {
			filter.Rating = &rVal
		} else {
			return response.Error(c, fiber.StatusBadRequest, "rating filter must be between 1 and 5", nil)
		}
	}

	if hasCommentStr := c.Query("has_comment"); hasCommentStr != "" {
		hasComment := hasCommentStr == "true" || hasCommentStr == "1"
		filter.HasComment = &hasComment
	}

	res, err := h.usecase.GetBookReviews(c.Context(), slug, filter, pq)
	if err != nil {
		if errors.Is(err, entity.ErrBookNotFound) {
			return response.Error(c, fiber.StatusNotFound, "book not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "book reviews retrieved successfully", res)
}

func (h *ReviewHandler) CheckOrderEligibility(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	rawInvoice := c.Params("invoice_number")
	invoiceNumber, err := url.QueryUnescape(rawInvoice)
	if err != nil {
		invoiceNumber = rawInvoice
	}
	invoiceNumber = strings.TrimSpace(invoiceNumber)

	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	eligibility, err := h.usecase.CheckOrderReviewEligibility(c.Context(), userID, invoiceNumber)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, "order not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "order review eligibility retrieved successfully", eligibility)
}

func (h *ReviewHandler) SubmitOrderReviews(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	rawInvoice := c.Params("invoice_number")
	invoiceNumber, err := url.QueryUnescape(rawInvoice)
	if err != nil {
		invoiceNumber = rawInvoice
	}
	invoiceNumber = strings.TrimSpace(invoiceNumber)

	if invoiceNumber == "" {
		return response.Error(c, fiber.StatusBadRequest, "invoice number is required", nil)
	}

	var req dto.SubmitOrderReviewsRequest
	if err := c.Bind().Body(&req); err != nil {
		// also support single review payload seamlessly: {"book_id": 1, "rating": 5, "comment": "..."}
		var singleReq dto.CreateReviewItemRequest
		if sErr := c.Bind().Body(&singleReq); sErr == nil && singleReq.BookID > 0 {
			req = dto.SubmitOrderReviewsRequest{
				Reviews: []dto.CreateReviewItemRequest{singleReq},
			}
		} else {
			return response.Error(c, fiber.StatusBadRequest, "invalid request body format", err.Error())
		}
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err.Error())
	}

	results, err := h.usecase.SubmitOrderReviews(c.Context(), userID, invoiceNumber, req)
	if err != nil {
		if errors.Is(err, entity.ErrOrderNotFound) {
			return response.Error(c, fiber.StatusNotFound, "order not found", nil)
		}
		if errors.Is(err, entity.ErrReviewNotEligible) ||
			errors.Is(err, entity.ErrRatingPeriodExpired) ||
			errors.Is(err, entity.ErrReviewAlreadySubmitted) ||
			errors.Is(err, entity.ErrInvalidRatingScore) ||
			errors.Is(err, entity.ErrBookNotInOrder) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "reviews submitted successfully", results)
}

func (h *ReviewHandler) UpdateReview(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	idParam := c.Params("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid review id", nil)
	}

	var req dto.UpdateReviewRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err.Error())
	}

	res, err := h.usecase.UpdateReview(c.Context(), userID, uint(reviewID), req)
	if err != nil {
		if errors.Is(err, entity.ErrReviewNotFound) {
			return response.Error(c, fiber.StatusNotFound, "review not found", nil)
		}
		if errors.Is(err, entity.ErrUnauthorizedReview) {
			return response.Error(c, fiber.StatusForbidden, err.Error(), nil)
		}
		if errors.Is(err, entity.ErrRatingPeriodExpired) || errors.Is(err, entity.ErrInvalidRatingScore) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "review updated successfully", res)
}

func (h *ReviewHandler) GetMyReviews(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}

	pq := pagination.FromFiber(c)
	items, meta, err := h.usecase.GetCustomerReviews(c.Context(), userID, pq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "user reviews retrieved successfully", items, meta)
}

func (h *ReviewHandler) AdminFindAllReviews(c fiber.Ctx) error {
	pq := pagination.FromFiber(c)

	filter := repository.ReviewFilter{
		Search: c.Query("search"),
		SortBy: strings.ToLower(strings.TrimSpace(c.Query("sort_by", "newest"))),
	}

	if bookIDStr := c.Query("book_id"); bookIDStr != "" {
		if bID, err := strconv.ParseUint(bookIDStr, 10, 32); err == nil && bID > 0 {
			b := uint(bID)
			filter.BookID = &b
		}
	}

	if ratingStr := c.Query("rating"); ratingStr != "" {
		if rVal, err := strconv.Atoi(ratingStr); err == nil && rVal >= 1 && rVal <= 5 {
			filter.Rating = &rVal
		}
	}

	switch c.Query("is_hidden") {
	case "true", "1":
		val := true
		filter.IsHidden = &val
	case "false", "0":
		val := false
		filter.IsHidden = &val
	}

	items, meta, err := h.usecase.AdminFindAllReviews(c.Context(), filter, pq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "admin reviews retrieved successfully", items, meta)
}

func (h *ReviewHandler) AdminModerateReview(c fiber.Ctx) error {
	idParam := c.Params("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid review id", nil)
	}

	var req dto.AdminModerateReviewRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err.Error())
	}

	if err := h.usecase.AdminModerateReview(c.Context(), uint(reviewID), req); err != nil {
		if errors.Is(err, entity.ErrReviewNotFound) {
			return response.Error(c, fiber.StatusNotFound, "review not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "review visibility updated successfully", nil)
}

func (h *ReviewHandler) AdminReplyReview(c fiber.Ctx) error {
	idParam := c.Params("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid review id", nil)
	}

	var req dto.AdminReplyReviewRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err.Error())
	}

	if err := h.usecase.AdminReplyReview(c.Context(), uint(reviewID), req); err != nil {
		if errors.Is(err, entity.ErrReviewNotFound) {
			return response.Error(c, fiber.StatusNotFound, "review not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "seller reply added successfully", nil)
}

func (h *ReviewHandler) AdminDeleteReview(c fiber.Ctx) error {
	idParam := c.Params("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid review id", nil)
	}

	if err := h.usecase.AdminDeleteReview(c.Context(), uint(reviewID)); err != nil {
		if errors.Is(err, entity.ErrReviewNotFound) {
			return response.Error(c, fiber.StatusNotFound, "review not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "review deleted successfully", nil)
}
