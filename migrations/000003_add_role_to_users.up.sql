ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'customer';

ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'customer'));
