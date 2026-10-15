CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO users (
    name, 
    email, 
    password, 
    role, 
    is_active, 
    created_at, 
    updated_at
) 
SELECT 
    'Admin',
    'laurent@gmail.com',
    crypt('@Laurent123', gen_salt('bf', 10)),
    'admin',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE email = 'laurent@gmail.com'
);