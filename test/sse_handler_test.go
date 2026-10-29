package test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/sse"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func setupSSEServer(t *testing.T, hub sse.Hub, jwtService jwt.JWTService) (string, func()) {
	app := fiber.New()
	api := app.Group("/api/v1")
	authMiddleware := middleware.AuthMiddleware(jwtService)
	handler.NewSSEHandler(api, hub, authMiddleware)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	go func() {
		_ = app.Listener(ln)
	}()

	baseURL := "http://" + ln.Addr().String()
	cleanup := func() {
		_ = ln.Close()
		_ = app.ShutdownWithTimeout(500 * time.Millisecond)
	}
	return baseURL, cleanup
}

func TestSSEHandler_StreamOrders(t *testing.T) {
	jwtSecret := "this-is-a-very-long-secret-key-for-testing-only-12345"
	jwtService := jwt.NewJWTService(jwtSecret)
	hub := sse.NewHub()
	authMiddleware := middleware.AuthMiddleware(jwtService)

	t.Run("unauthorized when no token provided", func(t *testing.T) {
		app := fiber.New()
		api := app.Group("/api/v1")
		handler.NewSSEHandler(api, hub, authMiddleware)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/sse/orders", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("unauthorized when invalid token provided in cookie", func(t *testing.T) {
		app := fiber.New()
		api := app.Group("/api/v1")
		handler.NewSSEHandler(api, hub, authMiddleware)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/sse/orders", nil)
		req.Header.Set("Cookie", "access_token=invalid.jwt.token")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("success connect with valid access_token cookie and receive live events", func(t *testing.T) {
		token, err := jwtService.GenerateToken(50, "customer@example.com", "customer")
		assert.NoError(t, err)

		baseURL, cleanup := setupSSEServer(t, hub, jwtService)
		defer cleanup()

		req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/sse/orders", nil)
		assert.NoError(t, err)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

		reader := bufio.NewReader(resp.Body)
		// Read initial connected event
		eventLine, err := reader.ReadString('\n')
		assert.NoError(t, err)
		assert.Equal(t, "event: connected\n", eventLine)

		// Publish payment update
		time.Sleep(20 * time.Millisecond)
		hub.PublishPaymentStatus(sse.PaymentStatusEvent{
			OrderID:       10,
			InvoiceNumber: "INV-50-PAID",
			UserID:        50,
			PaymentStatus: "settlement",
			OrderStatus:   "paid",
		})

		// Read payment_status event from SSE stream
		eventReceived := false
		for i := 0; i < 5; i++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			if strings.Contains(line, "event: payment_status") {
				eventReceived = true
				break
			}
		}
		assert.True(t, eventReceived, "client should receive published payment_status event")
	})

	t.Run("success connect with valid JWT bearer header and receive shipping event", func(t *testing.T) {
		token, err := jwtService.GenerateToken(1, "admin@example.com", "admin")
		assert.NoError(t, err)

		baseURL, cleanup := setupSSEServer(t, hub, jwtService)
		defer cleanup()

		req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/sse/orders", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

		reader := bufio.NewReader(resp.Body)
		// Read connected line
		_, _ = reader.ReadString('\n')

		// Publish shipping update
		time.Sleep(20 * time.Millisecond)
		hub.PublishShippingStatus(sse.ShippingStatusEvent{
			OrderID:        25,
			InvoiceNumber:  "INV-SHIP-25",
			UserID:         99,
			WaybillNumber:  "RESI-ABC-123",
			Courier:        "sicepat",
			CourierService: "sds",
			ShippingStatus: "in_transit",
			OrderStatus:    "shipped",
		})

		eventReceived := false
		for i := 0; i < 5; i++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			if strings.Contains(line, "event: shipping_status") {
				eventReceived = true
				break
			}
		}
		assert.True(t, eventReceived, "admin client should receive published shipping_status event")
	})
}
