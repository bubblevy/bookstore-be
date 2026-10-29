package sse

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/bubblevy/bookstore/pkg/logger"
)

// EventType defines the type of SSE event.
type EventType string

const (
	EventConnected      EventType = "connected"
	EventPaymentStatus  EventType = "payment_status"
	EventShippingStatus EventType = "shipping_status"
	EventPing           EventType = "ping"
)

// Event is the generic SSE payload sent to connected clients.
type Event struct {
	Type      EventType `json:"type"`
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// PaymentStatusEvent contains real-time payment update details.
type PaymentStatusEvent struct {
	OrderID       uint       `json:"order_id"`
	InvoiceNumber string     `json:"invoice_number"`
	UserID        uint       `json:"user_id"`
	PaymentStatus string     `json:"payment_status"`
	OrderStatus   string     `json:"order_status"`
	PaymentType   string     `json:"payment_type,omitempty"`
	GrossAmount   string     `json:"gross_amount,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ShippingStatusEvent contains real-time shipment & delivery update details.
type ShippingStatusEvent struct {
	OrderID         uint      `json:"order_id"`
	InvoiceNumber   string    `json:"invoice_number"`
	UserID          uint      `json:"user_id"`
	BiteshipOrderID string    `json:"biteship_order_id,omitempty"`
	TrackingID      string    `json:"tracking_id,omitempty"`
	WaybillNumber   string    `json:"waybill_number,omitempty"`
	Courier         string    `json:"courier,omitempty"`
	CourierService  string    `json:"courier_service,omitempty"`
	ShippingStatus  string    `json:"shipping_status"`
	OrderStatus     string    `json:"order_status,omitempty"`
	Note            string    `json:"note,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Hub defines the interface for managing SSE subscribers and broadcasting updates.
type Hub interface {
	PublishPaymentStatus(event PaymentStatusEvent)
	PublishShippingStatus(event ShippingStatusEvent)
	Subscribe(userID uint, role string, invoiceFilter string) (<-chan Event, func())
	ClientCount() int
}

type subscriber struct {
	id            string
	userID        uint
	role          string
	invoiceFilter string
	ch            chan Event
}

type hub struct {
	mu          sync.RWMutex
	subscribers map[string]*subscriber
}

// NewHub creates a new in-memory SSE event broker.
func NewHub() Hub {
	return &hub{
		subscribers: make(map[string]*subscriber),
	}
}

func (h *hub) Subscribe(userID uint, role string, invoiceFilter string) (<-chan Event, func()) {
	subID := generateRandomID()
	ch := make(chan Event, 64)

	sub := &subscriber{
		id:            subID,
		userID:        userID,
		role:          role,
		invoiceFilter: invoiceFilter,
		ch:            ch,
	}

	h.mu.Lock()
	h.subscribers[subID] = sub
	h.mu.Unlock()

	logger.Debug("sse client subscribed", "sub_id", subID, "user_id", userID, "role", role, "invoice", invoiceFilter)

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers, subID)
			h.mu.Unlock()

			close(ch)
			logger.Debug("sse client unsubscribed", "sub_id", subID, "user_id", userID)
		})
	}

	return ch, unsubscribe
}

func (h *hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}

func (h *hub) PublishPaymentStatus(event PaymentStatusEvent) {
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = time.Now()
	}

	sseEvt := Event{
		Type:      EventPaymentStatus,
		Data:      event,
		Timestamp: event.UpdatedAt,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.subscribers {
		if !h.matchesSubscriber(sub, event.UserID, event.InvoiceNumber) {
			continue
		}

		select {
		case sub.ch <- sseEvt:
		default:
			logger.Warn("sse subscriber channel full, dropping payment event", "sub_id", sub.id, "invoice", event.InvoiceNumber)
		}
	}
}

func (h *hub) PublishShippingStatus(event ShippingStatusEvent) {
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = time.Now()
	}

	sseEvt := Event{
		Type:      EventShippingStatus,
		Data:      event,
		Timestamp: event.UpdatedAt,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.subscribers {
		if !h.matchesSubscriber(sub, event.UserID, event.InvoiceNumber) {
			continue
		}

		select {
		case sub.ch <- sseEvt:
		default:
			logger.Warn("sse subscriber channel full, dropping shipping event", "sub_id", sub.id, "invoice", event.InvoiceNumber)
		}
	}
}

func (h *hub) matchesSubscriber(sub *subscriber, eventUserID uint, eventInvoice string) bool {
	// If subscriber specified an invoice filter, only match that invoice
	if sub.invoiceFilter != "" {
		return sub.invoiceFilter == eventInvoice
	}

	// Admin receives all events if no specific invoice filter is set
	if sub.role == "admin" {
		return true
	}

	// Regular user receives events for their own orders
	if sub.userID > 0 && sub.userID == eventUserID {
		return true
	}

	return false
}

func generateRandomID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}
