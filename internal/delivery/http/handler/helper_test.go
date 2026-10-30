package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestGetUserID(t *testing.T) {
	t.Run("returns uint user ID when present", func(t *testing.T) {
		app := fiber.New()
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("user_id", uint(42))
			uid, err := getUserID(c)
			assert.NoError(t, err)
			assert.Equal(t, uint(42), uid)
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("handles int user ID", func(t *testing.T) {
		app := fiber.New()
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("user_id", int(10))
			uid, err := getUserID(c)
			assert.NoError(t, err)
			assert.Equal(t, uint(10), uid)
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("handles float64 user ID", func(t *testing.T) {
		app := fiber.New()
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("user_id", float64(99))
			uid, err := getUserID(c)
			assert.NoError(t, err)
			assert.Equal(t, uint(99), uid)
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("returns unauthorized when user_id is missing", func(t *testing.T) {
		app := fiber.New()
		app.Get("/test", func(c fiber.Ctx) error {
			uid, err := getUserID(c)
			assert.Error(t, err)
			assert.Equal(t, uint(0), uid)
			assert.Equal(t, fiber.ErrUnauthorized, err)
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("returns unauthorized when user_id is non-positive or invalid type", func(t *testing.T) {
		app := fiber.New()
		app.Get("/test", func(c fiber.Ctx) error {
			c.Locals("user_id", "invalid-string")
			uid, err := getUserID(c)
			assert.Error(t, err)
			assert.Equal(t, uint(0), uid)
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}
