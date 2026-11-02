package middleware

import (
	"encoding/json"
	"strings"

	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/bubblevy/bookstore/pkg/turnstile"
	"github.com/gofiber/fiber/v3"
)

// Turnstile creates a Fiber middleware that validates Cloudflare Turnstile tokens from the request body.
// If verifier is nil or verifier.IsEnabled() is false, the middleware passes through without validation.
func Turnstile(verifier turnstile.Verifier) fiber.Handler {
	return func(c fiber.Ctx) error {
		if verifier == nil || !verifier.IsEnabled() {
			return c.Next()
		}

		var token string

		// Extract token strictly from JSON request body without consuming the stream (Fiber preserves c.Body() buffer)
		if len(c.Body()) > 0 {
			var bodyPayload struct {
				TurnstileToken      string `json:"turnstile_token"`
				CFTurnstileResponse string `json:"cf_turnstile_response"`
			}
			if err := json.Unmarshal(c.Body(), &bodyPayload); err == nil {
				if bodyPayload.TurnstileToken != "" {
					token = bodyPayload.TurnstileToken
				} else if bodyPayload.CFTurnstileResponse != "" {
					token = bodyPayload.CFTurnstileResponse
				}
			}
		}

		token = strings.TrimSpace(token)
		if token == "" {
			return response.Error(c, fiber.StatusBadRequest, "Cloudflare Turnstile token is required", nil)
		}

		// Resolve client IP (forwarded by Cloudflare proxy or reverse proxy)
		remoteIP := c.Get("CF-Connecting-IP")
		if remoteIP == "" {
			remoteIP = c.Get("X-Forwarded-For")
			if remoteIP != "" {
				if idx := strings.Index(remoteIP, ","); idx != -1 {
					remoteIP = strings.TrimSpace(remoteIP[:idx])
				}
			}
		}
		if remoteIP == "" {
			remoteIP = c.IP()
		}

		valid, err := verifier.Verify(c.Context(), token, remoteIP)
		if err != nil || !valid {
			logger.Warn("turnstile verification failed", "ip", remoteIP, "error", err)
			return response.Error(c, fiber.StatusBadRequest, "Cloudflare Turnstile verification failed or token is invalid", nil)
		}

		return c.Next()
	}
}
