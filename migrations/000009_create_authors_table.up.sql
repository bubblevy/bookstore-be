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
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();