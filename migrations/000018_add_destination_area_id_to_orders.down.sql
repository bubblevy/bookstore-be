DROP INDEX IF EXISTS idx_orders_destination_area_id;

ALTER TABLE orders
DROP COLUMN IF EXISTS destination_area_id;