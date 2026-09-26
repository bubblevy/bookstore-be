DROP INDEX IF EXISTS idx_users_activation_token;
DROP INDEX IF EXISTS idx_users_reset_password_token;

ALTER TABLE users 
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS activation_token,
    DROP COLUMN IF EXISTS activation_expires_at,
    DROP COLUMN IF EXISTS reset_password_token,
    DROP COLUMN IF EXISTS reset_password_expires_at;