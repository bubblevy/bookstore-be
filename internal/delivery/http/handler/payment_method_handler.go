package handler

import (
	"errors"
	"strconv"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/bubblevy/bookstore/pkg/storage"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type PaymentMethodHandler struct {
	usecase       usecase.PaymentMethodUsecase
	validate      *validator.Validate
	uploadUsecase usecase.UploadUsecase
}

func NewPaymentMethodHandler(
	r fiber.Router,
	uc usecase.PaymentMethodUsecase,
	v *validator.Validate,
	authMiddleware fiber.Handler,
	adminMiddleware fiber.Handler,
	uploadUC ...usecase.UploadUsecase,
) *PaymentMethodHandler {
	var uUC usecase.UploadUsecase
	if len(uploadUC) > 0 {
		uUC = uploadUC[0]
	}
	h := &PaymentMethodHandler{
		usecase:       uc,
		validate:      v,
		uploadUsecase: uUC,
	}

	setupRoutes := func(group fiber.Router) {
		// Public routes: GET list & GET detail
		group.Get("/", h.GetAll)
		group.Get("/:id", h.GetByID)

		// Protected routes: EDIT only (Admin / Owner)
		adminGroup := group.Group("", authMiddleware, adminMiddleware)
		adminGroup.Put("/:id", h.Update)
	}

	setupRoutes(r.Group("/payment-methods"))

	return h
}

func (h *PaymentMethodHandler) GetAll(c fiber.Ctx) error {
	amount, _ := strconv.Atoi(c.Query("amount", "0"))
	onlyActive := c.Query("only_active") == "true"

	res, err := h.usecase.GetAll(c.Context(), onlyActive, amount)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "payment methods retrieved successfully", res)
}

func (h *PaymentMethodHandler) GetByID(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid ID parameter", nil)
	}

	res, err := h.usecase.GetByID(c.Context(), uint(id))
	if err != nil {
		if errors.Is(err, entity.ErrPaymentMethodNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "payment method retrieved successfully", res)
}

func (h *PaymentMethodHandler) Update(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid ID parameter", nil)
	}

	var req dto.UpdatePaymentMethodRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	// Handle direct multipart file upload if provided (key: 'image' or 'file')
	fileHeader, err := c.FormFile("image")
	if err != nil {
		fileHeader, err = c.FormFile("file")
	}
	if err == nil && fileHeader != nil && h.uploadUsecase != nil {
		fileReader, err := fileHeader.Open()
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "failed to open uploaded file", nil)
		}
		defer fileReader.Close()

		uploadRes, err := h.uploadUsecase.UploadImage(c.Context(), fileReader, fileHeader.Filename, fileHeader.Size, "payments")
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrInvalidFileType),
				errors.Is(err, storage.ErrFileTooLarge),
				errors.Is(err, storage.ErrEmptyFile):
				return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
			default:
				return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
			}
		}
		req.Image = &uploadRes.FileURL
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation error", response.FormatValidationErrors(err))
	}

	res, err := h.usecase.Update(c.Context(), uint(id), req)
	if err != nil {
		if errors.Is(err, entity.ErrPaymentMethodNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, entity.ErrPaymentMethodCodeImmutable) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "payment method updated successfully", res)
}
