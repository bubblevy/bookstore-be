package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/bubblevy/bookstore/config"
	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/cookie"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	u              usecase.AuthUsecase
	validate       *validator.Validate
	authMiddleware fiber.Handler
	oauthConfig    *oauth2.Config
	frontendURL    string
}

func NewAuthHandler(r fiber.Router, u usecase.AuthUsecase, v *validator.Validate, authMiddleware fiber.Handler, cfg config.AppConfig) {
	h := &AuthHandler{
		u:              u,
		validate:       v,
		authMiddleware: authMiddleware,
		oauthConfig:    cfg.GoogleOAuth,
		frontendURL:    cfg.SMTP.FrontendURL,
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
	r.Get("/google/login", h.GoogleLogin)
	r.Get("/google/callback", h.GoogleCallback)
	r.Post("/google/mobile", middleware.RateLimit(middleware.StrictRateLimit), h.GoogleMobileLogin)
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
		if errors.Is(err, entity.ErrAccountAlreadyActive) {
			return response.Success(c, fiber.StatusOK, "account is already active", nil)
		}
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
	return response.Success(c, fiber.StatusOK, "login successful", dto.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
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
	return response.Success(c, fiber.StatusOK, "token refreshed successfully", dto.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	refreshToken := cookie.GetRefreshToken(c)

	if refreshToken == "" {
		var req dto.RefreshTokenRequest
		if err := c.Bind().Body(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

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
	if currentRefreshToken == "" && req.RefreshToken != "" {
		currentRefreshToken = req.RefreshToken
	}

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

func (h *AuthHandler) GoogleLogin(c fiber.Ctx) error {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   300,
		HTTPOnly: true,
	})

	url := h.oauthConfig.AuthCodeURL(state)
	return c.Redirect().To(url)
}

func (h *AuthHandler) GoogleCallback(c fiber.Ctx) error {
	state := c.Query("state")
	cookieState := c.Cookies("oauth_state")

	if state != cookieState {
		return response.Error(c, fiber.StatusUnauthorized, "invalid oauth state", nil)
	}

	c.Cookie(&fiber.Cookie{Name: "oauth_state", MaxAge: -1})

	code := c.Query("code")
	if code == "" {
		return response.Error(c, fiber.StatusBadRequest, "code not found", nil)
	}

	result, err := h.u.GoogleLogin(c.Context(), code, h.oauthConfig)
	if err != nil {
		return c.Redirect().To(fmt.Sprintf("%s/login?error=oauth_failed", h.frontendURL))
	}

	cookie.SetAuthCookies(c, result.AccessToken, result.RefreshToken)
	return c.Redirect().To(fmt.Sprintf("%s/dashboard", h.frontendURL))
}

func (h *AuthHandler) GoogleMobileLogin(c fiber.Ctx) error {
	var req dto.GoogleMobileLoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if req.IDToken == "" && req.Code == "" {
		return response.Error(c, fiber.StatusBadRequest, "either id_token or code is required", nil)
	}

	result, err := h.u.GoogleMobileLogin(c.Context(), req.IDToken, req.Code, h.oauthConfig)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	cookie.SetAuthCookies(c, result.AccessToken, result.RefreshToken)
	return response.Success(c, fiber.StatusOK, "google login successful", dto.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

