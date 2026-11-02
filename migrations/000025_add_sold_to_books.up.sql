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

