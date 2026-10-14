-- Create a native PostgreSQL ENUM type
CREATE TYPE order_status AS ENUM (
    'pending',
    'processing',
    'shipped',
    'completed',
    'cancelled'
);

CREATE TYPE shipping_status AS ENUM (
    'pending',
    'picked_up',
    'in_transit',
    'delivered'
);

-- Orders Table
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    order_code VARCHAR(50) UNIQUE NOT NULL, -- Order ID Midtrans
    user_id BIGINT NOT NULL,
    subtotal NUMERIC(12, 2) NOT NULL CHECK (subtotal >= 0),
    shipping_cost NUMERIC(12, 2) NOT NULL CHECK (shipping_cost >= 0),
    total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
    status order_status DEFAULT 'pending' NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Order Items Table (Snapshot of price and weight at checkout)
CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    book_id BIGINT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    price NUMERIC(12, 2) NOT NULL CHECK (price >= 0),
    weight_grams INT NOT NULL CHECK (weight_grams > 0),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_order_items_book FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE RESTRICT
);

-- Shipping Table (Shipments RajaOngkir)
CREATE TABLE IF NOT EXISTS shipments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT UNIQUE NOT NULL,
    courier VARCHAR(50) NOT NULL,          -- jne, tiki, pos, sicepat
    courier_service VARCHAR(50) NOT NULL,  -- REG, YES, OKE, CTC, dll.
    etd VARCHAR(30),                       -- Estimated arrival (eg: "2-3 DAYS")
    total_weight INT NOT NULL CHECK (total_weight > 0), -- Total gram
    tracking_number VARCHAR(100),          -- Receipt number
    status shipping_status DEFAULT 'pending' NOT NULL,
    origin_city_id INT NOT NULL,           -- City ID of shop/warehouse
    destination_city_id INT NOT NULL,      -- ID of the customer's destination city
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_shipments_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_book_id ON order_items(book_id);
CREATE INDEX IF NOT EXISTS idx_shipments_order_id ON shipments(order_id);