package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/bubblevy/bookstore/pkg/sse"
	"github.com/gofiber/fiber/v3"
)

type SSEHandler struct {
	hub sse.Hub
}

func NewSSEHandler(r fiber.Router, hub sse.Hub, authMiddleware fiber.Handler) *SSEHandler {
	h := &SSEHandler{hub: hub}

	r.Get("/sse/orders", authMiddleware, h.StreamOrders)

	return h
}

func (h *SSEHandler) StreamOrders(c fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	userRole, _ := c.Locals("user_role").(string)

	invoiceFilter := strings.TrimSpace(c.Query("invoice"))

	// Set standard SSE Headers
	c.Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Set("Cache-Control", "no-cache, no-transform")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no") // Prevents proxy/Nginx buffering

	// Subscribe ke hub
	events, unsubscribe := h.hub.Subscribe(userID, userRole, invoiceFilter)

	logger.Info("sse client connected to order stream",
		"user_id", userID,
		"role", userRole,
		"invoice", invoiceFilter,
		"ip", c.IP(),
	)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer unsubscribe()

		// Send initial connected handshake event
		connectPayload, _ := json.Marshal(map[string]any{
			"status":    "connected",
			"user_id":   userID,
			"invoice":   invoiceFilter,
			"timestamp": time.Now().UTC(),
		})
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", sse.EventConnected, string(connectPayload)); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		// Keep-alive heartbeat ticker every 15 seconds
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		ctx := c.Context()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				// Send comment keep-alive line to maintain connection through firewalls/proxies
				if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}

			case evt, ok := <-events:
				if !ok {
					return
				}

				dataBytes, err := json.Marshal(evt.Data)
				if err != nil {
					logger.Error("failed marshaling sse event payload", "error", err)
					continue
				}

				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, string(dataBytes)); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}
