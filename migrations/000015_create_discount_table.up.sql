-- Tabel Discounts
CREATE TABLE IF NOT EXISTS discounts (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('PERCENTAGE', 'FIXED')),
    discount_value NUMERIC(12, 2) NOT NULL CHECK (discount_value > 0),
    min_purchase NUMERIC(12, 2) DEFAULT 0 NOT NULL CHECK (min_purchase >= 0),
    max_discount_amount NUMERIC(12, 2) CHECK (max_discount_amount > 0),
    quota INT DEFAULT 0 NOT NULL CHECK (quota >= 0),
    used_count INT DEFAULT 0 NOT NULL CHECK (used_count >= 0),
    is_all_books BOOLEAN DEFAULT TRUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_discount_dates CHECK (end_date > start_date)
);

-- Partial Unique Index untuk kode voucher aktif (soft delete safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_discounts_code_active 
    ON discounts(code) WHERE deleted_at IS NULL;

-- Index filter tanggal & status aktif
CREATE INDEX IF NOT EXISTS idx_discounts_active_dates 
    ON discounts(is_active, start_date, end_date) WHERE deleted_at IS NULL;

-- Pivot Table Many-to-Many: discount_books
CREATE TABLE IF NOT EXISTS discount_books (
    discount_id BIGINT NOT NULL,
    book_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (discount_id, book_id),
    CONSTRAINT fk_discount_books_discount FOREIGN KEY (discount_id) 
        REFERENCES discounts(id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_discount_books_book FOREIGN KEY (book_id) 
        REFERENCES books(id) ON UPDATE CASCADE ON DELETE CASCADE
);

-- Index foreign key untuk performa join/lookup per buku
CREATE INDEX IF NOT EXISTS idx_discount_books_book_id ON discount_books(book_id);