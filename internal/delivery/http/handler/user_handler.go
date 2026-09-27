package handler

import (
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/cookie"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	u              usecase.AuthUsecase
	validate       *validator.Validate
	authMiddleware fiber.Handler
}

func NewAuthHandler(r fiber.Router, u usecase.AuthUsecase, validate *validator.Validate, authMiddleware fiber.Handler) {
	h := &AuthHandler{
		u:              u,
		validate:       validate,
		authMiddleware: authMiddleware,
	}

	// routes
	r.Get("/me", h.authMiddleware, h.GetProfile)

	auth := r.Group("/auth")
	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
	auth.Post("/refresh-token", h.RefreshToken)
	// auth.Post("/reset-password", h.ResetPassword)
	// auth.Post("/change-password", h.ChangePassword)
	auth.Delete("/logout", h.Logout)
	// auth.Put("/update-profile", h.UpdateProfile)
	auth.Post("/resend-activation", h.ResendActivation)
	auth.Get("/activate-account", h.ActivateAccount)
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

	if err := h.u.Register(c.Context(), req.Name, req.Email, req.Password); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "User registered successfully. Please check your email to activate your account.", nil)
}

func (h *AuthHandler) ResendActivation(c fiber.Ctx) error {
	var req entity.ResendActivationRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		errorsMap := response.FormatValidationErrors(err)
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", errorsMap)
	}

	if err := h.u.ResendActivation(c.Context(), req.Email); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to process request", nil)
	}

	return response.Success(c, fiber.StatusOK, "If your email is registered and not yet activated, a new activation link has been sent to your inbox.", nil)
}

func (h *AuthHandler) ActivateAccount(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return response.Error(c, fiber.StatusBadRequest, "Activation token is required", nil)
	}

	if err := h.u.ActivateAccount(c.Context(), token); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Account activated successfully. You can now login.", nil)
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

	// set cookie
	cookie.SetAuthCookies(c, token.AccessToken, token.RefreshToken)
	return response.Success(c, fiber.StatusOK, "Login successful", nil)
}

func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var refreshToken string
	refreshToken = c.Cookies("refresh_token")

	if refreshToken == "" {
		var req entity.RefreshTokenRequest
		if err := c.Bind().Body(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Refresh token is missing", nil)
	}

	token, err := h.u.RefreshToken(c.Context(), refreshToken)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	// set cookie
	cookie.SetAuthCookies(c, token.AccessToken, token.RefreshToken)
	return response.Success(c, fiber.StatusOK, "Token refreshed successfully", nil)
}

func (h *AuthHandler) GetProfile(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized access", nil)
	}

	user, err := h.u.GetProfile(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Profile retrieved successfully", user)
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	cookie.ClearAuthCookies(c)
	return response.Success(c, fiber.StatusOK, "Logout successful", nil)
}
