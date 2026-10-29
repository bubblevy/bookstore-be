-- Performance composite indexes for orders querying and sorting
CREATE INDEX IF NOT EXISTS idx_orders_user_created ON orders(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at DESC);

-- Performance composite indexes for books catalog filtering & sorting (popular, price)
CREATE INDEX IF NOT EXISTS idx_books_active_sold ON books(is_active, sold DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_books_active_price ON books(is_active, price) WHERE deleted_at IS NULL;

-- Performance indexes for payments and shipments status lookups
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_shipments_status ON shipments(status);
