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
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();