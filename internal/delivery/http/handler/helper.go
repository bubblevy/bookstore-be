package handler

import "github.com/gofiber/fiber/v3"

// getUserID extracts the authenticated user's ID from fiber context locals.
// It safely handles various numeric types (uint, int, float64) stored by middleware or tests.
func getUserID(c fiber.Ctx) (uint, error) {
	val := c.Locals("user_id")
	if val == nil {
		return 0, fiber.ErrUnauthorized
	}
	switch v := val.(type) {
	case uint:
		return v, nil
	case int:
		if v <= 0 {
			return 0, fiber.ErrUnauthorized
		}
		return uint(v), nil
	case float64:
		if v <= 0 {
			return 0, fiber.ErrUnauthorized
		}
		return uint(v), nil
	default:
		return 0, fiber.ErrUnauthorized
	}
}
