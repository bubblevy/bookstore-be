package test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/bubblevy/bookstore/config"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}
	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

// ============================================================================
// MOCK AUTH USECASE
// ============================================================================

type MockAuthUsecase struct {
	mock.Mock
}

func (m *MockAuthUsecase) Register(ctx context.Context, input usecase.RegisterInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockAuthUsecase) ResendActivation(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockAuthUsecase) ActivateAccount(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockAuthUsecase) Login(ctx context.Context, input usecase.LoginInput) (*usecase.AuthResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthResult), args.Error(1)
}

func (m *MockAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*usecase.AuthResult, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthResult), args.Error(1)
}

func (m *MockAuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

func (m *MockAuthUsecase) ForgotPassword(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockAuthUsecase) ResetPassword(ctx context.Context, input usecase.ResetPasswordInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockAuthUsecase) ChangePassword(ctx context.Context, input usecase.ChangePasswordInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockAuthUsecase) GoogleLogin(ctx context.Context, code string, cfg *oauth2.Config) (*usecase.AuthResult, error) {
	args := m.Called(ctx, code, cfg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthResult), args.Error(1)
}

func (m *MockAuthUsecase) StartSessionCleanup(ctx context.Context, interval time.Duration) {
	m.Called(ctx, interval)
}

// ============================================================================
// SETUP HELPER
// ============================================================================

func setupAuthApp(mockUC *MockAuthUsecase, authenticated bool) (*fiber.App, config.AppConfig) {
	app := fiber.New()
	validate := validator.New()

	_ = validate.RegisterValidation("strong_password", validateStrongPassword)

	authMiddleware := func(c fiber.Ctx) error {
		if !authenticated {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    401,
				"status":  false,
				"message": "unauthorized",
			})
		}
		c.Locals("user_id", uint(1))
		return c.Next()
	}

	appCfg := config.AppConfig{
		GoogleOAuth: &oauth2.Config{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  "http://localhost:8080/api/v1/auth/google/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		},
	}
	appCfg.SMTP.FrontendURL = "http://frontend.local"

	api := app.Group("/api/v1/auth")
	handler.NewAuthHandler(api, mockUC, validate, authMiddleware, appCfg)

	return app, appCfg
}

// ============================================================================
// REGISTER & ACTIVATION TESTS
// ============================================================================

func TestAuthHandler_Register_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	expectedInput := usecase.RegisterInput{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "Password123!",
	}

	mockUC.On("Register", mock.Anything, expectedInput).Return(nil)

	payload := `{"name":"Test User","email":"test@example.com","password":"Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	payload := `{"name":"","email":"invalid-email","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
}

func TestAuthHandler_ResendActivation_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	mockUC.On("ResendActivation", mock.Anything, "test@example.com").Return(nil)

	payload := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-activation", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_ActivateAccount_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	mockUC.On("ActivateAccount", mock.Anything, "valid-token").Return(nil)

	payload := `{"token":"valid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/activate-account", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_ActivateAccount_AlreadyActive(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	mockUC.On("ActivateAccount", mock.Anything, "active-token").Return(entity.ErrAccountAlreadyActive)

	payload := `{"token":"active-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/activate-account", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

// ============================================================================
// LOGIN, REFRESH TOKEN, & LOGOUT TESTS
// ============================================================================

func TestAuthHandler_Login_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	input := usecase.LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	}

	result := &usecase.AuthResult{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
	}

	mockUC.On("Login", mock.Anything, input).Return(result, nil)

	payload := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	input := usecase.LoginInput{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	mockUC.On("Login", mock.Anything, input).Return(nil, errors.New("invalid credentials"))

	payload := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken_FromCookie_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	result := &usecase.AuthResult{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}

	mockUC.On("RefreshToken", mock.Anything, "existing-cookie-refresh-token").Return(result, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh-token", nil)
	req.Header.Set("Cookie", "refresh_token=existing-cookie-refresh-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken_Missing(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh-token", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUC.AssertNotCalled(t, "RefreshToken", mock.Anything, mock.Anything)
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, true)

	mockUC.On("Logout", mock.Anything, "active-refresh-token").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/logout", nil)
	req.Header.Set("Cookie", "refresh_token=active-refresh-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

// ============================================================================
// PASSWORD MANAGEMENT TESTS
// ============================================================================

func TestAuthHandler_ForgotPassword_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	mockUC.On("ForgotPassword", mock.Anything, "test@example.com").Return(nil)

	payload := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	input := usecase.ResetPasswordInput{
		Token:           "reset-token-123",
		NewPassword:     "NewSecret123!",
		ConfirmPassword: "NewSecret123!",
	}

	mockUC.On("ResetPassword", mock.Anything, input).Return(nil)

	payload := `{"token":"reset-token-123","new_password":"NewSecret123!","confirm_password":"NewSecret123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, true)

	input := usecase.ChangePasswordInput{
		UserID:              1,
		CurrentPassword:     "OldSecret123!",
		NewPassword:         "NewSecret123!",
		ConfirmPassword:     "NewSecret123!",
		CurrentRefreshToken: "user-refresh-token",
	}

	mockUC.On("ChangePassword", mock.Anything, input).Return(nil)

	payload := `{"current_password":"OldSecret123!","new_password":"NewSecret123!","confirm_password":"NewSecret123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "refresh_token=user-refresh-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_ChangePassword_Unauthorized(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	payload := `{"current_password":"oldSecret123","new_password":"newSecret123","confirm_password":"newSecret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUC.AssertNotCalled(t, "ChangePassword", mock.Anything, mock.Anything)
}

// ============================================================================
// GOOGLE OAUTH TESTS
// ============================================================================

func TestAuthHandler_GoogleLogin_Redirect(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusSeeOther, resp.StatusCode) // Ganti dari http.StatusFound (302) ke 303

	location := resp.Header.Get("Location")
	assert.Contains(t, location, "accounts.google.com")
	assert.Contains(t, location, "state=")

	setCookieHeader := resp.Header.Get("Set-Cookie")
	assert.Contains(t, setCookieHeader, "oauth_state=")
}

func TestAuthHandler_GoogleCallback_InvalidState(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=bad-state&code=test-code", nil)
	req.Header.Set("Cookie", "oauth_state=real-state")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUC.AssertNotCalled(t, "GoogleLogin", mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthHandler_GoogleCallback_MissingCode(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, _ := setupAuthApp(mockUC, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=match-state", nil)
	req.Header.Set("Cookie", "oauth_state=match-state")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertNotCalled(t, "GoogleLogin", mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthHandler_GoogleCallback_Success(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, cfg := setupAuthApp(mockUC, false)

	mockResult := &usecase.AuthResult{
		AccessToken:  "google-access-token",
		RefreshToken: "google-refresh-token",
	}

	mockUC.On("GoogleLogin", mock.Anything, "valid-oauth-code", cfg.GoogleOAuth).Return(mockResult, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=match-state&code=valid-oauth-code", nil)
	req.Header.Set("Cookie", "oauth_state=match-state")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusSeeOther, resp.StatusCode) // Ganti dari http.StatusFound (302) ke 303

	expectedRedirect := fmt.Sprintf("%s/dashboard", cfg.SMTP.FrontendURL)
	assert.Equal(t, expectedRedirect, resp.Header.Get("Location"))
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_GoogleCallback_UsecaseFailure(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	app, cfg := setupAuthApp(mockUC, false)

	mockUC.On("GoogleLogin", mock.Anything, "failed-code", cfg.GoogleOAuth).Return(nil, errors.New("exchange token error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=match-state&code=failed-code", nil)
	req.Header.Set("Cookie", "oauth_state=match-state")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusSeeOther, resp.StatusCode) // Ganti dari http.StatusFound (302) ke 303

	expectedRedirect := fmt.Sprintf("%s/login?error=oauth_failed", cfg.SMTP.FrontendURL)
	assert.Equal(t, expectedRedirect, resp.Header.Get("Location"))
	mockUC.AssertExpectations(t)
}
