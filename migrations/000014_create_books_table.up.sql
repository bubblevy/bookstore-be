CREATE TABLE IF NOT EXISTS books (
    id BIGSERIAL PRIMARY KEY,
    category_id INT NOT NULL,
    author_id INT NOT NULL,
    publisher_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(270) NOT NULL,
    isbn VARCHAR(30),
    synopsis TEXT,
    price NUMERIC(12, 2) NOT NULL CHECK (price >= 0),
    weight_grams INT NOT NULL CHECK (weight_grams > 0),
    pages INT CHECK (pages > 0),
    publication_year SMALLINT CHECK (publication_year >= 1000 AND publication_year <= 2100),
    stock INT DEFAULT 0 NOT NULL CHECK (stock >= 0),
    cover_image VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_books_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
    CONSTRAINT fk_books_author FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE RESTRICT,
    CONSTRAINT fk_books_publisher FOREIGN KEY (publisher_id) REFERENCES publishers(id) ON DELETE RESTRICT
);

-- Unique constraints for active records
CREATE UNIQUE INDEX IF NOT EXISTS uq_books_slug_active 
    ON books(slug) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_books_isbn_active 
    ON books(isbn) WHERE deleted_at IS NULL AND isbn IS NOT NULL;

-- Relational foreign key indexes
CREATE INDEX IF NOT EXISTS idx_books_category_id ON books(category_id);
CREATE INDEX IF NOT EXISTS idx_books_author_id ON books(author_id);
CREATE INDEX IF NOT EXISTS idx_books_publisher_id ON books(publisher_id);
CREATE INDEX IF NOT EXISTS idx_books_deleted_at ON books(deleted_at);

-- Composite index for query katalog toko/pencarian utama
CREATE INDEX IF NOT EXISTS idx_books_active_created 
    ON books(is_active, created_at DESC) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_books_updated_at ON books;
CREATE TRIGGER trg_books_updated_at 
    BEFORE UPDATE ON books 
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();