ALTER TABLE orders
ADD COLUMN IF NOT EXISTS destination_area_id VARCHAR(100) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_orders_destination_area_id ON orders(destination_area_id);