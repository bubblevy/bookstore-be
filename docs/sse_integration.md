# Server-Sent Events (SSE) Real-Time Order & Shipping Integration

This document outlines the Server-Sent Events (SSE) specification and integration guidelines for client applications to receive real-time order payment status updates (Midtrans) and shipping tracking updates (Biteship) without polling.

---

## 1. Endpoint Specification

- **Endpoint URL**: `GET /api/v1/sse/orders`
- **Response Format**: `text/event-stream; charset=utf-8`
- **Protocol**: HTTP/1.1 or HTTP/2 Server-Sent Events
- **Response Headers**:
  - `Content-Type: text/event-stream; charset=utf-8`
  - `Cache-Control: no-cache, no-transform`
  - `Connection: keep-alive`
  - `Transfer-Encoding: chunked`
  - `X-Accel-Buffering: no` *(prevents reverse proxies like Nginx from buffering event streams)*

---

## 2. Authentication & Authorization

The endpoint is protected by `AuthMiddleware`. Every incoming connection must provide valid authentication credentials; otherwise, the server rejects the request with an HTTP `401 Unauthorized` status code.

Two authentication mechanisms are supported:

### Method 1: HTTP-Only Cookie (Browser Native `EventSource`)
The standard W3C browser `EventSource` API **does not support custom HTTP request headers** (such as `Authorization: Bearer <token>`). When using native `EventSource`, authentication is performed via the `access_token` HTTP-only cookie by setting `{ withCredentials: true }`:

```javascript
// The browser automatically includes the access_token cookie when withCredentials is true
const eventSource = new EventSource('/api/v1/sse/orders', {
  withCredentials: true,
});
```

> **Note**: The user must be authenticated beforehand so that the `access_token` cookie is present on the client.

### Method 2: HTTP Header `Authorization: Bearer <token>`
If your Single Page Application (SPA) stores the JWT in memory or `localStorage` without using cookies, use an SSE client library such as [`@microsoft/fetch-event-source`](https://www.npmjs.com/package/@microsoft/fetch-event-source) to provide standard `Bearer` authorization headers:

```typescript
import { fetchEventSource } from '@microsoft/fetch-event-source';

fetchEventSource('/api/v1/sse/orders', {
  method: 'GET',
  headers: {
    'Authorization': `Bearer ${token}`,
  },
  // event handlers...
});
```

---

## 3. Query Parameters

| Parameter | Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `invoice` | `string` | Optional | The specific invoice number to subscribe to (e.g., `INV-20261029-0001`). If specified, the stream only delivers events related to this invoice. If omitted, the stream delivers all updates for orders belonging to the authenticated `user_id` (or all system orders if the user has the `admin` role). |

Example request filtered by invoice:
```
GET /api/v1/sse/orders?invoice=INV-20261029-0001
```

---

## 4. Event Types & Schemas

### A. Event `connected`
Dispatched by the server immediately after an SSE connection is authorized and registered with the subscription hub.

**Payload Schema**:
```json
{
  "status": "connected",
  "user_id": 42,
  "invoice": "INV-20261029-0001",
  "timestamp": "2026-10-29T10:15:00Z"
}
```

---

### B. Event `payment_status`
Triggered in real-time when the Midtrans payment gateway webhook notifies the server of an order transaction state change (e.g., settlement, expiration, or cancellation).

**Payload Schema**:
```json
{
  "order_id": 105,
  "invoice_number": "INV-20261029-0001",
  "user_id": 42,
  "payment_status": "settlement",
  "order_status": "paid",
  "payment_type": "bank_transfer",
  "gross_amount": "150000.00",
  "paid_at": "2026-10-29T10:15:20Z",
  "updated_at": "2026-10-29T10:15:21Z"
}
```

- **`payment_status` values**: `settlement`, `pending`, `deny`, `expire`, `cancel`
- **`order_status` values**: `paid`, `waiting_payment`, `cancelled`

---

### C. Event `shipping_status`
Triggered when an administrator schedules courier pickup (Biteship), when the courier updates live tracking checkpoints (via Biteship webhooks), or when a return shipment is processed.

**Payload Schema**:
```json
{
  "order_id": 105,
  "invoice_number": "INV-20261029-0001",
  "user_id": 42,
  "biteship_order_id": "6720e8b123...",
  "tracking_id": "track_12345",
  "waybill_number": "TK1234567890",
  "courier": "jne",
  "courier_service": "reg",
  "shipping_status": "in_transit",
  "order_status": "shipped",
  "note": "Package is in transit to destination hub",
  "updated_at": "2026-10-29T10:16:00Z"
}
```

- **`shipping_status` values**: `allocated`, `picking_up`, `picked`, `dropping_off`, `in_transit`, `delivered`, `return_in_transit`, `returned`, `cancelled`
- **`order_status` values**: `processing`, `shipped`, `completed`, `returned`

---

### D. Keep-Alive Heartbeat
The server sends a comment ping every 15 seconds:
```
: keepalive
```
This prevents intermediary network equipment, load balancers, Cloudflare/Nginx reverse proxies, and browsers from timing out idle connections when no events are actively being published.

---

## 5. Client Implementation Guide

### Option 1: Native Browser `EventSource` (Cookie-Based)

Recommended when the application relies on secure HTTP-only cookies. No external dependencies required.

```javascript
const invoiceNumber = 'INV-20261029-0001';

// Construct the SSE URL with an optional invoice filter
const sseUrl = invoiceNumber 
  ? `http://localhost:8000/api/v1/sse/orders?invoice=${encodeURIComponent(invoiceNumber)}`
  : 'http://localhost:8000/api/v1/sse/orders';

// Setting withCredentials to true is mandatory for sending the access_token cookie
const eventSource = new EventSource(sseUrl, { withCredentials: true });

// 1. Initial connection handshake
eventSource.addEventListener('connected', (event) => {
  const data = JSON.parse(event.data);
  console.log('SSE connection established:', data);
});

// 2. Real-time payment updates (Midtrans)
eventSource.addEventListener('payment_status', (event) => {
  const data = JSON.parse(event.data);
  console.log('Payment status update received:', data);

  if (data.payment_status === 'settlement' || data.order_status === 'paid') {
    alert(`Payment successful for order ${data.invoice_number}!`);
    // Trigger UI updates or refetch order details
  } else if (data.order_status === 'cancelled') {
    alert(`Order ${data.invoice_number} has been cancelled or expired.`);
  }
});

// 3. Real-time courier shipping updates (Biteship)
eventSource.addEventListener('shipping_status', (event) => {
  const data = JSON.parse(event.data);
  console.log('Shipping status update received:', data);

  if (data.shipping_status === 'delivered') {
    alert(`Package for order ${data.invoice_number} has been delivered!`);
  }
});

// 4. Connection error handling
eventSource.onerror = (err) => {
  console.error('SSE connection interrupted or failed:', err);
  // Native EventSource automatically handles reconnection backoff
};

// 5. Cleanup upon component unmount or route change
// eventSource.close();
```

---

### Option 2: `@microsoft/fetch-event-source` (Bearer Token Header)

Recommended for Single Page Applications (React, Vue, Angular, Svelte) storing JWT tokens in memory or `localStorage`:

1. Install the official package:
   ```bash
   npm install @microsoft/fetch-event-source
   ```

2. Establish the connection:
   ```typescript
   import { fetchEventSource } from '@microsoft/fetch-event-source';

   const token = localStorage.getItem('access_token');
   const invoiceNumber = 'INV-20261029-0001';
   const url = `http://localhost:8000/api/v1/sse/orders?invoice=${encodeURIComponent(invoiceNumber)}`;

   const abortController = new AbortController();

   fetchEventSource(url, {
     method: 'GET',
     headers: {
       'Authorization': `Bearer ${token}`,
     },
     signal: abortController.signal,
     async onopen(response) {
       if (response.ok && response.headers.get('content-type')?.includes('text/event-stream')) {
         console.log('SSE connection established');
         return;
       }
       throw new Error(`Failed to establish SSE stream. Status: ${response.status}`);
     },
     onmessage(msg) {
       if (msg.event === 'connected') {
         console.log('Handshake event:', JSON.parse(msg.data));
       } else if (msg.event === 'payment_status') {
         const data = JSON.parse(msg.data);
         console.log('Payment event received:', data);
       } else if (msg.event === 'shipping_status') {
         const data = JSON.parse(msg.data);
         console.log('Shipping event received:', data);
       }
     },
     onerror(err) {
       console.error('SSE stream error:', err);
     },
   });

   // To disconnect when navigating away:
   // abortController.abort();
   ```

---

### Option 3: Production React Hook (`useOrderStatusSSE.ts`)

A reusable React hook supporting native cookies with auto-cleanup:

```tsx
import { useEffect, useState } from 'react';

export interface PaymentEvent {
  order_id: number;
  invoice_number: string;
  user_id: number;
  payment_status: string;
  order_status: string;
  payment_type?: string;
  gross_amount?: string;
  paid_at?: string;
  updated_at: string;
}

export interface ShippingEvent {
  order_id: number;
  invoice_number: string;
  user_id: number;
  biteship_order_id?: string;
  tracking_id?: string;
  waybill_number?: string;
  courier?: string;
  courier_service?: string;
  shipping_status: string;
  order_status?: string;
  note?: string;
  updated_at: string;
}

interface UseOrderStatusSSEProps {
  invoice?: string;
  enabled?: boolean;
}

export function useOrderStatusSSE({ invoice, enabled = true }: UseOrderStatusSSEProps = {}) {
  const [paymentUpdate, setPaymentUpdate] = useState<PaymentEvent | null>(null);
  const [shippingUpdate, setShippingUpdate] = useState<ShippingEvent | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!enabled) return;

    const params = new URLSearchParams();
    if (invoice) params.set('invoice', invoice);

    const queryString = params.toString() ? `?${params.toString()}` : '';
    const sseUrl = `http://localhost:8000/api/v1/sse/orders${queryString}`;

    const es = new EventSource(sseUrl, {
      withCredentials: true,
    });

    es.addEventListener('connected', () => {
      setIsConnected(true);
    });

    es.addEventListener('payment_status', (e) => {
      try {
        const data: PaymentEvent = JSON.parse(e.data);
        setPaymentUpdate(data);
      } catch (err) {
        console.error('Failed to parse payment_status event:', err);
      }
    });

    es.addEventListener('shipping_status', (e) => {
      try {
        const data: ShippingEvent = JSON.parse(e.data);
        setShippingUpdate(data);
      } catch (err) {
        console.error('Failed to parse shipping_status event:', err);
      }
    });

    es.onerror = () => {
      setIsConnected(false);
    };

    return () => {
      es.close();
      setIsConnected(false);
    };
  }, [invoice, enabled]);

  return { isConnected, paymentUpdate, shippingUpdate };
}
```

---

## 6. Security & Architectural Considerations

1. **Token Protection**: JWT credentials are never transmitted via query strings (`?token=`), preventing credential leakage into browser histories, server access logs, and upstream proxy logs.
2. **CORS Headers**: The backend CORS middleware explicitly permits `Last-Event-ID`, `Cache-Control`, and `Connection`, with `AllowCredentials: true` enabled.
3. **Role-Based Event Scoping**: Customers (`role: customer`) can only receive events pertaining to their own orders. Administrative users (`role: admin`) can monitor transactions store-wide or filter by specific invoice numbers.
