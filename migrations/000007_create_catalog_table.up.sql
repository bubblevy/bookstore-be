CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_categories_slug_active 
    ON categories(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_categories_deleted_at 
    ON categories(deleted_at);

DROP TRIGGER IF EXISTS trg_categories_updated_at ON categories;
CREATE TRIGGER trg_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS authors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(170) NOT NULL,
    bio TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_authors_slug_active 
    ON authors(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_authors_deleted_at 
    ON authors(deleted_at);

DROP TRIGGER IF EXISTS trg_authors_updated_at ON authors;
CREATE TRIGGER trg_authors_updated_at
    BEFORE UPDATE ON authors
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS publishers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(170) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_publishers_slug_active 
    ON publishers(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_publishers_deleted_at 
    ON publishers(deleted_at);

DROP TRIGGER IF EXISTS trg_publishers_updated_at ON publishers;
CREATE TRIGGER trg_publishers_updated_at
    BEFORE UPDATE ON publishers
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

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
    
    CONSTRAINT fk_books_category FOREIGN KEY (category_id) 
        REFERENCES categories(id) ON DELETE RESTRICT,
    CONSTRAINT fk_books_author FOREIGN KEY (author_id) 
        REFERENCES authors(id) ON DELETE RESTRICT,
    CONSTRAINT fk_books_publisher FOREIGN KEY (publisher_id) 
        REFERENCES publishers(id) ON DELETE RESTRICT
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
    ON books(is_active, created_at DESC) 
    WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_books_updated_at ON books;
CREATE TRIGGER trg_books_updated_at
    BEFORE UPDATE ON books
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();