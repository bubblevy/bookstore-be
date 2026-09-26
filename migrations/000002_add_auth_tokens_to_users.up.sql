ALTER TABLE users 
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS activation_token VARCHAR(255),
    ADD COLUMN IF NOT EXISTS activation_expires_at TIMESTAMPTZ,
  
    ADD COLUMN IF NOT EXISTS reset_password_token VARCHAR(255),
    ADD COLUMN IF NOT EXISTS reset_password_expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_activation_token ON users(activation_token);
CREATE INDEX IF NOT EXISTS idx_users_reset_password_token ON users(reset_password_token);