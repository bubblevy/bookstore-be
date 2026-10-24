ALTER TABLE books ADD COLUMN IF NOT EXISTS sold INT DEFAULT 0 NOT NULL CHECK (sold >= 0);

-- Backfill sold count from existing order items for paid/completed orders
UPDATE books b
SET sold = COALESCE((
    SELECT SUM(oi.quantity)
    FROM order_items oi
    JOIN orders o ON o.id = oi.order_id
    WHERE oi.book_id = b.id
      AND o.status IN ('paid', 'processing', 'shipped', 'completed')
), 0);

CREATE INDEX IF NOT EXISTS idx_books_sold ON books(sold DESC);

-- Trigger to automatically update books.sold when order status changes
CREATE OR REPLACE FUNCTION update_books_sold_on_order_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- If status changed to revenue eligible (paid, processing, shipped, completed) from a non-eligible status
    IF (NEW.status IN ('paid', 'processing', 'shipped', 'completed')) AND 
       (OLD.status NOT IN ('paid', 'processing', 'shipped', 'completed')) THEN
        UPDATE books b
        SET sold = b.sold + oi.total_qty
        FROM (
            SELECT book_id, SUM(quantity) as total_qty
            FROM order_items
            WHERE order_id = NEW.id
            GROUP BY book_id
        ) oi
        WHERE b.id = oi.book_id;

    -- If status changed from revenue eligible to non-eligible (cancelled, returned)
    ELSIF (OLD.status IN ('paid', 'processing', 'shipped', 'completed')) AND 
          (NEW.status NOT IN ('paid', 'processing', 'shipped', 'completed')) THEN
        UPDATE books b
        SET sold = GREATEST(0, b.sold - oi.total_qty)
        FROM (
            SELECT book_id, SUM(quantity) as total_qty
            FROM order_items
            WHERE order_id = NEW.id
            GROUP BY book_id
        ) oi
        WHERE b.id = oi.book_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_orders_update_books_sold ON orders;
CREATE TRIGGER trg_orders_update_books_sold
    AFTER UPDATE OF status ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_books_sold_on_order_status_change();
