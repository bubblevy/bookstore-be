package handler

import (
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	u        usecase.AuthUsecase
	validate *validator.Validate
}

func NewAuthHandler(r fiber.Router, u usecase.AuthUsecase, validate *validator.Validate) {
	h := &AuthHandler{
		u:        u,
		validate: validate,
	}

	auth := r.Group("/auth")
	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req entity.RegisterRequest

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		errorsMap := response.FormatValidationErrors(err)
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", errorsMap)
	}

	token, err := h.u.Register(c.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "User registered successfully", entity.AuthResponse{
		Token: token,
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req entity.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		errorsMap := response.FormatValidationErrors(err)
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", errorsMap)
	}

	token, err := h.u.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Login successful", entity.AuthResponse{
		Token: token,
	})
}
