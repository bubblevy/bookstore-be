package handler

import (
	"errors"
	"strings"

	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/bubblevy/bookstore/pkg/storage"
	"github.com/gofiber/fiber/v3"
)

type UploadHandler struct {
	usecase usecase.UploadUsecase
}

func NewUploadHandler(r fiber.Router, uc usecase.UploadUsecase, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &UploadHandler{usecase: uc}

	uploads := r.Group("/uploads", authMiddleware, adminMiddleware)
	uploads.Post("/", h.Upload)
	uploads.Delete("/", h.Delete)
}

func (h *UploadHandler) Upload(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "file is required in multipart form data (key: 'file')", nil)
	}

	folder := c.FormValue("folder")
	if folder == "" {
		folder = c.Query("folder", "uploads")
	}

	fileReader, err := fileHeader.Open()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to open uploaded file", nil)
	}
	defer fileReader.Close()

	res, err := h.usecase.UploadImage(c.Context(), fileReader, fileHeader.Filename, fileHeader.Size, folder)
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

	return response.Success(c, fiber.StatusCreated, "file uploaded successfully", res)
}

func (h *UploadHandler) Delete(c fiber.Ctx) error {
	fileURL := strings.TrimSpace(c.Query("file_url"))
	fileKey := strings.TrimSpace(c.Query("file_key"))

	target := fileURL
	if target == "" {
		target = fileKey
	}

	if target == "" {
		return response.Error(c, fiber.StatusBadRequest, "file_url or file_key query parameter is required", nil)
	}

	if err := h.usecase.DeleteFile(c.Context(), target); err != nil {
		if errors.Is(err, storage.ErrFileNotFound) {
			return response.Error(c, fiber.StatusNotFound, "file not found in storage", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to delete file", nil)
	}

	return response.Success(c, fiber.StatusOK, "file deleted successfully", nil)
}
