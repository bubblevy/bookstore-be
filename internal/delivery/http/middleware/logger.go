package middleware

import (
	"time"

	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/gofiber/fiber/v3"
)

// HTTPLogger logs every incoming HTTP request with method, path, status, latency, and client IP
func HTTPLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// Execute downstream handlers
		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()

		// Exclude healthcheck routes from verbose logging
		if path == "/health" || path == "/health/ready" {
			return err
		}

		requestID, _ := c.Locals("request_id").(string)
		ip := c.IP()

		fields := []any{
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"ip", ip,
		}

		if requestID != "" {
			fields = append(fields, "request_id", requestID)
		}

		if err != nil {
			fields = append(fields, "error", err.Error())
		}

		if status >= 500 {
			logger.Error("http server error", fields...)
		} else if status >= 400 {
			logger.Warn("http client error", fields...)
		} else {
			logger.Info("http request", fields...)
		}

		return err
	}
}
