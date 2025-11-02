CREATE TYPE user_role AS ENUM ('user', 'moderator', 'admin');
CREATE TYPE user_state AS ENUM ('active', 'blocked', 'suspended');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    email VARCHAR(320) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL DEFAULT 'user',
    state user_state NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);