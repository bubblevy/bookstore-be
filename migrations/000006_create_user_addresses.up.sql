CREATE TABLE IF NOT EXISTS user_addresses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    receiver_name VARCHAR(150) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    address_line VARCHAR(255) NOT NULL,
    destination_id INT NOT NULL,
    province_name VARCHAR(100) NOT NULL,
    city_name VARCHAR(100) NOT NULL,
    district_name VARCHAR(100) NOT NULL,
    subdistrict_name VARCHAR(100),
    postal_code VARCHAR(10) NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_user_addresses_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_user_addresses_destination_id ON user_addresses(destination_id);
CREATE INDEX IF NOT EXISTS idx_user_addresses_deleted_at ON user_addresses(deleted_at);