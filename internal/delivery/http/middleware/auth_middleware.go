package middleware

import (
	"errors"
	"strings"

	j "github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserID  uint
	Email   string
	Role    string
	IsAdmin bool
}

// extractTokenClaims extracts, validates, and parses JWT claims from Authorization header or cookie.
func extractTokenClaims(c fiber.Ctx, jwtService j.JWTService) (*TokenClaims, error) {
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
		return nil, errors.New("missing authorization token")
	}

	token, err := jwtService.ValidateToken(tokenString)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to parse token claims")
	}

	var userID uint
	if val, exists := claims["user_id"]; exists {
		if floatVal, ok := val.(float64); ok {
			userID = uint(floatVal)
		}
	}

	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)

	if userID == 0 || email == "" || role == "" {
		return nil, errors.New("incomplete token payload")
	}

	return &TokenClaims{
		UserID:  userID,
		Email:   email,
		Role:    role,
		IsAdmin: role == "admin",
	}, nil
}

// AuthMiddleware validates authentication token.
// When optional is true, it populates user locals if a valid token is provided,
// but allows unauthenticated requests to continue without error.
func AuthMiddleware(jwtService j.JWTService, optional ...bool) fiber.Handler {
	isOptional := len(optional) > 0 && optional[0]

	return func(c fiber.Ctx) error {
		claims, err := extractTokenClaims(c, jwtService)
		if err != nil {
			if isOptional {
				return c.Next()
			}
			return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)
		c.Locals("is_admin", claims.IsAdmin)

		return c.Next()
	}
}
