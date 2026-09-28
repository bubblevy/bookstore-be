package middleware

import (
	"time"

	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

type RateLimitConfig struct {
	Max        int
	Expiration time.Duration
	KeyPrefix  string
}

var defaultRateLimitConfig = RateLimitConfig{
	Max:        100,
	Expiration: 1 * time.Minute,
	KeyPrefix:  "default",
}

func RateLimit(cfg RateLimitConfig) fiber.Handler {
	if cfg.Max == 0 {
		cfg.Max = defaultRateLimitConfig.Max
	}
	if cfg.Expiration == 0 {
		cfg.Expiration = defaultRateLimitConfig.Expiration
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = defaultRateLimitConfig.KeyPrefix
	}

	return limiter.New(limiter.Config{
		Max:        cfg.Max,
		Expiration: cfg.Expiration,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP() + ":" + cfg.KeyPrefix
		},
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests, "Too many requests", nil)
		},
	})
}

var (
	AuthRateLimit = RateLimitConfig{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyPrefix:  "auth",
	}

	StandardRateLimit = RateLimitConfig{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyPrefix:  "standard",
	}

	ReadRateLimit = RateLimitConfig{
		Max:        200,
		Expiration: 1 * time.Minute,
		KeyPrefix:  "read",
	}
)
