package sse_test

import (
	"sync"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/pkg/sse"
	"github.com/stretchr/testify/assert"
)

func TestSSEHub_PublishPaymentStatus(t *testing.T) {
	hub := sse.NewHub()

	t.Run("user receives own order payment update", func(t *testing.T) {
		events, unsubscribe := hub.Subscribe(10, "customer", "")
		defer unsubscribe()

		hub.PublishPaymentStatus(sse.PaymentStatusEvent{
			OrderID:       1,
			InvoiceNumber: "INV-1001",
			UserID:        10,
			PaymentStatus: "settlement",
			OrderStatus:   "paid",
		})

		select {
		case evt := <-events:
			assert.Equal(t, sse.EventPaymentStatus, evt.Type)
			payData, ok := evt.Data.(sse.PaymentStatusEvent)
			assert.True(t, ok)
			assert.Equal(t, "INV-1001", payData.InvoiceNumber)
			assert.Equal(t, "settlement", payData.PaymentStatus)
			assert.Equal(t, "paid", payData.OrderStatus)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timed out waiting for payment status event")
		}
	})

	t.Run("user does not receive other users order payment update", func(t *testing.T) {
		events, unsubscribe := hub.Subscribe(20, "customer", "")
		defer unsubscribe()

		hub.PublishPaymentStatus(sse.PaymentStatusEvent{
			OrderID:       2,
			InvoiceNumber: "INV-1002",
			UserID:        10, // Different user
			PaymentStatus: "settlement",
			OrderStatus:   "paid",
		})

		select {
		case <-events:
			t.Fatal("unexpected event received for different user")
		case <-time.After(100 * time.Millisecond):
			// Success: no event received
		}
	})

	t.Run("admin receives all payment updates", func(t *testing.T) {
		events, unsubscribe := hub.Subscribe(999, "admin", "")
		defer unsubscribe()

		hub.PublishPaymentStatus(sse.PaymentStatusEvent{
			OrderID:       3,
			InvoiceNumber: "INV-1003",
			UserID:        42,
			PaymentStatus: "expire",
			OrderStatus:   "cancelled",
		})

		select {
		case evt := <-events:
			assert.Equal(t, sse.EventPaymentStatus, evt.Type)
			payData, ok := evt.Data.(sse.PaymentStatusEvent)
			assert.True(t, ok)
			assert.Equal(t, "INV-1003", payData.InvoiceNumber)
			assert.Equal(t, "expire", payData.PaymentStatus)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("admin did not receive event")
		}
	})

	t.Run("specific invoice filter matches", func(t *testing.T) {
		events, unsubscribe := hub.Subscribe(0, "guest", "INV-SPECIFIC")
		defer unsubscribe()

		// Send non-matching
		hub.PublishPaymentStatus(sse.PaymentStatusEvent{
			OrderID:       4,
			InvoiceNumber: "INV-OTHER",
			UserID:        10,
			PaymentStatus: "settlement",
		})

		// Send matching
		hub.PublishPaymentStatus(sse.PaymentStatusEvent{
			OrderID:       5,
			InvoiceNumber: "INV-SPECIFIC",
			UserID:        10,
			PaymentStatus: "settlement",
			OrderStatus:   "paid",
		})

		select {
		case evt := <-events:
			payData := evt.Data.(sse.PaymentStatusEvent)
			assert.Equal(t, "INV-SPECIFIC", payData.InvoiceNumber)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timed out waiting for specific invoice payment event")
		}
	})
}

func TestSSEHub_PublishShippingStatus(t *testing.T) {
	hub := sse.NewHub()

	t.Run("user receives shipping status update", func(t *testing.T) {
		events, unsubscribe := hub.Subscribe(15, "customer", "")
		defer unsubscribe()

		hub.PublishShippingStatus(sse.ShippingStatusEvent{
			OrderID:        10,
			InvoiceNumber:  "INV-SHIP-1",
			UserID:         15,
			WaybillNumber:  "RESI12345",
			Courier:        "jne",
			CourierService: "reg",
			ShippingStatus: "in_transit",
			OrderStatus:    "shipped",
		})

		select {
		case evt := <-events:
			assert.Equal(t, sse.EventShippingStatus, evt.Type)
			shipData, ok := evt.Data.(sse.ShippingStatusEvent)
			assert.True(t, ok)
			assert.Equal(t, "INV-SHIP-1", shipData.InvoiceNumber)
			assert.Equal(t, "in_transit", shipData.ShippingStatus)
			assert.Equal(t, "shipped", shipData.OrderStatus)
			assert.Equal(t, "RESI12345", shipData.WaybillNumber)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timed out waiting for shipping status event")
		}
	})
}

func TestSSEHub_Concurrency(t *testing.T) {
	hub := sse.NewHub()
	var wg sync.WaitGroup

	for i := 1; i <= 20; i++ {
		wg.Add(1)
		go func(uid uint) {
			defer wg.Done()
			events, unsubscribe := hub.Subscribe(uid, "customer", "")
			defer unsubscribe()

			hub.PublishPaymentStatus(sse.PaymentStatusEvent{
				OrderID:       uid,
				InvoiceNumber: "INV",
				UserID:        uid,
				PaymentStatus: "settlement",
			})

			select {
			case <-events:
			case <-time.After(1 * time.Second):
			}
		}(uint(i))
	}

	wg.Wait()
}
