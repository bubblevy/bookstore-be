package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/pkg/turnstile"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTurnstileVerifier struct {
	enabled    bool
	siteKey    string
	verifyFunc func(ctx context.Context, token string, remoteIP string) (bool, error)
}

var _ turnstile.Verifier = (*mockTurnstileVerifier)(nil)

func (m *mockTurnstileVerifier) Verify(ctx context.Context, token string, remoteIP string) (bool, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, token, remoteIP)
	}
	if !m.enabled {
		return true, nil
	}
	if token == "valid-token" {
		return true, nil
	}
	return false, errors.New("invalid turnstile token")
}

func (m *mockTurnstileVerifier) IsEnabled() bool {
	return m.enabled
}

func (m *mockTurnstileVerifier) GetSiteKey() string {
	return m.siteKey
}

func TestTurnstileMiddleware_Disabled(t *testing.T) {
	app := fiber.New()
	mockVerifier := &mockTurnstileVerifier{enabled: false}
	app.Post("/test", middleware.Turnstile(mockVerifier), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTurnstileMiddleware_MissingToken(t *testing.T) {
	app := fiber.New()
	mockVerifier := &mockTurnstileVerifier{enabled: true}
	app.Post("/test", middleware.Turnstile(mockVerifier), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestTurnstileMiddleware_HeaderOrQueryIgnored_RequiresBody(t *testing.T) {
	app := fiber.New()
	mockVerifier := &mockTurnstileVerifier{enabled: true}
	app.Post("/test", middleware.Turnstile(mockVerifier), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Header alone should NOT be accepted
	reqHeader := httptest.NewRequest(http.MethodPost, "/test", nil)
	reqHeader.Header.Set("CF-Turnstile-Response", "valid-token")
	respHeader, err := app.Test(reqHeader)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respHeader.StatusCode)

	// Query parameter alone should NOT be accepted
	reqQuery := httptest.NewRequest(http.MethodPost, "/test?turnstile_token=valid-token", nil)
	respQuery, err := app.Test(reqQuery)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respQuery.StatusCode)
}

func TestTurnstileMiddleware_JsonBody_TurnstileToken_Success(t *testing.T) {
	app := fiber.New()
	mockVerifier := &mockTurnstileVerifier{enabled: true}

	type TestReq struct {
		Email          string `json:"email"`
		TurnstileToken string `json:"turnstile_token"`
	}

	app.Post("/test", middleware.Turnstile(mockVerifier), func(c fiber.Ctx) error {
		var req TestReq
		if err := c.Bind().Body(&req); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		assert.Equal(t, "user@example.com", req.Email)
		return c.SendStatus(fiber.StatusOK)
	})

	body, _ := json.Marshal(TestReq{
		Email:          "user@example.com",
		TurnstileToken: "valid-token",
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTurnstileMiddleware_JsonBody_CFTurnstileResponse_Success(t *testing.T) {
	app := fiber.New()
	mockVerifier := &mockTurnstileVerifier{enabled: true}

	body, _ := json.Marshal(map[string]string{
		"cf_turnstile_response": "valid-token",
	})

	app.Post("/test", middleware.Turnstile(mockVerifier), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTurnstileMiddleware_InvalidTokenInBody_Rejected(t *testing.T) {
	app := fiber.New()
	mockVerifier := &mockTurnstileVerifier{enabled: true}
	app.Post("/test", middleware.Turnstile(mockVerifier), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	body, _ := json.Marshal(map[string]string{
		"turnstile_token": "wrong-token",
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestTurnstileMiddleware_ClientIPForwarded(t *testing.T) {
	app := fiber.New()
	var capturedIP string
	mockVerifier := &mockTurnstileVerifier{
		enabled: true,
		verifyFunc: func(ctx context.Context, token string, remoteIP string) (bool, error) {
			capturedIP = remoteIP
			return true, nil
		},
	}

	app.Post("/test", middleware.Turnstile(mockVerifier), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	body, _ := json.Marshal(map[string]string{
		"turnstile_token": "any-token",
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CF-Connecting-IP", "203.0.113.195")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "203.0.113.195", capturedIP)
}
