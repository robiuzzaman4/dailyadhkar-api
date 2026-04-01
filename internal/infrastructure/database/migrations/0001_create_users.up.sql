CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    is_subscribed BOOLEAN NOT NULL DEFAULT TRUE,
    total_email_received INTEGER NOT NULL DEFAULT 0,
    role TEXT NOT NULL DEFAULT 'user',
    gender TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_role_check CHECK (role IN ('user', 'admin')),
    CONSTRAINT users_gender_check CHECK (gender IN ('male', 'female')),
    CONSTRAINT users_total_email_received_check CHECK (total_email_received >= 0)
);

CREATE INDEX IF NOT EXISTS idx_users_is_subscribed ON users (is_subscribed);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
