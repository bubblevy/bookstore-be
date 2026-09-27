package middleware

import (
	"strings"

	j "github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtService j.JWTService) fiber.Handler {
	return func(c fiber.Ctx) error {
		var tokenString string
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = c.Cookies("access_token")
		}

		if tokenString == "" {
			return response.Error(c, fiber.StatusUnauthorized, "missing authorization token", nil)
		}

		token, err := jwtService.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			return response.Error(c, fiber.StatusUnauthorized, "invalid or expired token", nil)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.Error(c, fiber.StatusUnauthorized, "failed to parse token claims", nil)
		}

		var userID uint
		if val, exists := claims["user_id"]; exists {
			if floatVal, ok := val.(float64); ok {
				userID = uint(floatVal)
			}
		}

		email, _ := claims["email"].(string)

		if userID == 0 || email == "" {
			return response.Error(c, fiber.StatusUnauthorized, "incomplete token payload", nil)
		}

		role, ok := claims["role"].(string)
		if !ok || role == "" {
			return response.Error(c, fiber.StatusUnauthorized, "incomplete token payload", nil)
		}

		c.Locals("user_id", userID)
		c.Locals("user_email", email)
		c.Locals("user_role", role)

		return c.Next()
	}
}
