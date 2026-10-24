DROP TRIGGER IF EXISTS trg_orders_update_books_sold ON orders;
DROP FUNCTION IF EXISTS update_books_sold_on_order_status_change();
DROP INDEX IF EXISTS idx_books_sold;
ALTER TABLE books DROP COLUMN IF EXISTS sold;
