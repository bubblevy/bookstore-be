CREATE TYPE shipping_status AS ENUM (
    'placed',
    'scheduled',
    'confirmed',
    'allocated',
    'picking_up',
    'picked',
    'dropping_off',
    'in_transit',
    'delivered',
    'rejected',
    'courier_not_found',
    'cancelled',
    'returned',
    'on_hold',
    'disposed',
    'return_in_transit'
);

CREATE TABLE IF NOT EXISTS shipments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL UNIQUE REFERENCES orders(id) ON DELETE RESTRICT,
    biteship_order_id VARCHAR(100) NOT NULL UNIQUE,
    biteship_tracking_id VARCHAR(100),
    waybill_number VARCHAR(100),
    courier VARCHAR(50) NOT NULL,
    courier_service VARCHAR(50) NOT NULL,
    status shipping_status NOT NULL DEFAULT 'allocated',
    tracking_url TEXT,
    order_note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shipments_order_id ON shipments(order_id);
CREATE INDEX IF NOT EXISTS idx_shipments_biteship_order_id ON shipments(biteship_order_id);
CREATE INDEX IF NOT EXISTS idx_shipments_waybill ON shipments(waybill_number);