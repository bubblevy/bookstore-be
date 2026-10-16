package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
)

func BiteshipWebhookAuth(expectedSignature string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if expectedSignature == "" {
			return response.Error(c, fiber.StatusUnauthorized, "webhook signature is not configured", nil)
		}

		signature := strings.TrimSpace(c.Get("X-Biteship-Signature"))
		if signature == "" {
			return response.Error(c, fiber.StatusUnauthorized, "missing X-Biteship-Signature header", nil)
		}

		if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
			return response.Error(c, fiber.StatusUnauthorized, "invalid webhook signature", nil)
		}

		return c.Next()
	}
}
