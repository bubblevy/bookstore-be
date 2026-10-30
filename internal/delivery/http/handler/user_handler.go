package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type UserHandler struct {
	usecase usecase.UserUsecase
}

func NewUserHandler(r fiber.Router, uc usecase.UserUsecase, authMiddleware fiber.Handler, adminMiddleware fiber.Handler) {
	h := &UserHandler{
		usecase: uc,
	}

	users := r.Group("/users")

	// Customer profile endpoint (safe, non-sensitive)
	users.Get("/me", authMiddleware, h.GetMe)

	// Admin user management with pagination
	users.Get("/", authMiddleware, adminMiddleware, h.FindAll)
}

// GetMe retrieves the authenticated user's profile with non-sensitive fields only.
func (h *UserHandler) GetMe(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized access", nil)
	}

	profile, err := h.usecase.GetProfile(c.Context(), userID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return response.Error(c, fiber.StatusNotFound, "user profile not found", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "user profile retrieved successfully", profile)
}

// FindAll lists users with pagination for admin management.
func (h *UserHandler) FindAll(c fiber.Ctx) error {
	pq := pagination.FromFiber(c)

	var isActive *bool
	if activeQuery := strings.TrimSpace(c.Query("is_active")); activeQuery != "" {
		if b, err := strconv.ParseBool(activeQuery); err == nil {
			isActive = &b
		}
	}

	filter := repository.UserFilter{
		Search:   strings.TrimSpace(c.Query("search")),
		Role:     strings.TrimSpace(c.Query("role")),
		IsActive: isActive,
	}

	items, meta, err := h.usecase.FindAll(c.Context(), filter, pq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.SuccessWithPagination(c, fiber.StatusOK, "users retrieved successfully", items, meta)
}
