package middleware

import (
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
)

func AdminMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		role, ok := c.Locals("user_role").(string)
		if !ok || role != "admin" {
			return response.Error(c, fiber.StatusForbidden, "access denied: insufficient permissions", nil)
		}

		return c.Next()
	}
}
