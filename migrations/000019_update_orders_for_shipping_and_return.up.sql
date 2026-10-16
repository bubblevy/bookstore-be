-- Add nilai 'returned' ke enum order_status yang sudah ada
ALTER TYPE order_status ADD VALUE IF NOT EXISTS 'returned';

-- snapshot area biteship dan audit retur ke tabel orders
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS returned_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS return_reason TEXT;