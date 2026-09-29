package handler

import (
	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
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
	r.Post("/register", middleware.RateLimit(middleware.StandardRateLimit), h.Register)
	r.Post("/login", middleware.RateLimit(middleware.StrictRateLimit), h.Login)
	r.Post("/refresh-token", middleware.RateLimit(middleware.StandardRateLimit), h.RefreshToken)
	r.Post("/forgot-password", middleware.RateLimit(middleware.StrictRateLimit), h.ForgotPassword)
	r.Post("/reset-password", middleware.RateLimit(middleware.StrictRateLimit), h.ResetPassword)
	r.Post("/change-password", authMiddleware, middleware.RateLimit(middleware.StrictRateLimit), h.ChangePassword)
	r.Delete("/logout", middleware.RateLimit(middleware.StandardRateLimit), h.Logout)
	r.Post("/resend-activation", middleware.RateLimit(middleware.StandardRateLimit), h.ResendActivation)
	r.Post("/activate-account", middleware.RateLimit(middleware.StandardRateLimit), h.ActivateAccount)
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req dto.RegisterRequest

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	i := usecase.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.u.Register(c.Context(), i); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "user registered successfully. please check your email to activate your account.", nil)
}

func (h *AuthHandler) ResendActivation(c fiber.Ctx) error {
	var req dto.ResendActivationRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	if err := h.u.ResendActivation(c.Context(), req.Email); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to process request", nil)
	}

	return response.Success(c, fiber.StatusOK, "if your email is registered and not yet activated, a new activation link has been sent to your inbox", nil)
}

func (h *AuthHandler) ActivateAccount(c fiber.Ctx) error {
	var req dto.ActivateAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	if err := h.u.ActivateAccount(c.Context(), req.Token); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "account activated successfully", nil)
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	i := usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	result, err := h.u.Login(c.Context(), i)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	// set cookie
	cookie.SetAuthCookies(c, result.AccessToken, result.RefreshToken)
	return response.Success(c, fiber.StatusOK, "login successful", nil)
}

func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var refreshToken string
	refreshToken = c.Cookies("refresh_token")

	if refreshToken == "" {
		var req dto.RefreshTokenRequest
		if err := c.Bind().Body(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		return response.Error(c, fiber.StatusUnauthorized, "refresh token is missing", nil)
	}

	result, err := h.u.RefreshToken(c.Context(), refreshToken)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	// set cookie
	cookie.SetAuthCookies(c, result.AccessToken, result.RefreshToken)
	return response.Success(c, fiber.StatusOK, "token refreshed successfully", nil)
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	refreshToken := cookie.GetRefreshToken(c)

	if refreshToken != "" {
		_ = h.u.Logout(c.Context(), refreshToken)
	}

	cookie.ClearAuthCookies(c)
	return response.Success(c, fiber.StatusOK, "logout successful", nil)
}

func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var req dto.ForgotPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	if err := h.u.ForgotPassword(c.Context(), req.Email); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to process request", nil)
	}

	return response.Success(c, fiber.StatusOK, "if your email is registered, a password reset link has been sent to your inbox", nil)
}

func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	i := usecase.ResetPasswordInput{
		Token:           req.Token,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.ConfirmPassword,
	}

	if err := h.u.ResetPassword(c.Context(), i); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "reset password successfully", nil)
}

func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	var req dto.ChangePasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	userID := c.Locals("user_id").(uint)
	currentRefreshToken := cookie.GetRefreshToken(c)

	i := usecase.ChangePasswordInput{
		UserID:              userID,
		CurrentPassword:     req.CurrentPassword,
		NewPassword:         req.NewPassword,
		ConfirmPassword:     req.ConfirmPassword,
		CurrentRefreshToken: currentRefreshToken,
	}

	if err := h.u.ChangePassword(c.Context(), i); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "password changed successfully", nil)
}
