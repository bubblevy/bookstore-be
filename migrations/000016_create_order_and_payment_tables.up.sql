-- Enum Status
CREATE TYPE order_status AS ENUM (
    'waiting_payment',
    'paid',
    'processing',
    'shipped',
    'completed',
    'cancelled'
);

CREATE TYPE payment_status AS ENUM (
    'pending',
    'settlement',
    'expire',
    'cancel',
    'deny',
    'refund'
);

-- Tabel Orders
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    invoice_number VARCHAR(60) NOT NULL UNIQUE,
    status order_status NOT NULL DEFAULT 'waiting_payment',

    -- Rincian Biaya
    items_subtotal NUMERIC(15, 2) NOT NULL CHECK (items_subtotal >= 0),
    shipping_cost NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (shipping_cost >= 0),
    discount_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    admin_fee NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (admin_fee >= 0),
    total_amount NUMERIC(15, 2) NOT NULL CHECK (total_amount >= 0),

    -- Snapshot Diskon
    discount_id BIGINT REFERENCES discounts(id) ON DELETE SET NULL,
    discount_code VARCHAR(50),

    -- Detail Pengiriman & Snapshot Alamat
    courier VARCHAR(30) NOT NULL,
    courier_service VARCHAR(50) NOT NULL,
    total_weight_grams INT NOT NULL CHECK (total_weight_grams > 0),
    shipping_tracking_number VARCHAR(100),
    recipient_name VARCHAR(150) NOT NULL,
    recipient_phone VARCHAR(30) NOT NULL,
    recipient_address TEXT NOT NULL,
    province_name VARCHAR(100) NOT NULL,
    city_name VARCHAR(100) NOT NULL,
    district_name VARCHAR(100) NOT NULL,
    subdistrict_name VARCHAR(100) DEFAULT '',
    postal_code VARCHAR(10) NOT NULL,

    -- Waktu
    payment_deadline TIMESTAMP WITH TIME ZONE NOT NULL,
    paid_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancellation_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_invoice ON orders(invoice_number);

-- Tabel Order Items
CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    book_id BIGINT NOT NULL REFERENCES books(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    cover_image VARCHAR(255),
    price NUMERIC(15, 2) NOT NULL CHECK (price >= 0),
    weight_grams INT NOT NULL CHECK (weight_grams > 0),
    quantity INT NOT NULL CHECK (quantity > 0),
    subtotal NUMERIC(15, 2) NOT NULL CHECK (subtotal >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_book_id ON order_items(book_id);

-- Tabel Payments
CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    midtrans_transaction_id VARCHAR(100),
    payment_type VARCHAR(50) NOT NULL,
    bank VARCHAR(30),
    va_number VARCHAR(100),
    bill_key VARCHAR(100),
    biller_code VARCHAR(100),
    qr_url TEXT,
    status payment_status NOT NULL DEFAULT 'pending',
    raw_response JSONB,
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_midtrans_transaction_id ON payments(midtrans_transaction_id);