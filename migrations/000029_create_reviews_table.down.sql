DROP TABLE IF EXISTS reviews;

DROP INDEX IF EXISTS idx_books_average_rating;

ALTER TABLE books
DROP COLUMN IF EXISTS average_rating,
DROP COLUMN IF EXISTS total_reviews;

ALTER TABLE orders
DROP COLUMN IF EXISTS completed_at;

