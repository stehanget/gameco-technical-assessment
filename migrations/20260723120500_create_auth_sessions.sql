-- +goose Up
CREATE TABLE IF NOT EXISTS auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    jti VARCHAR(64) NOT NULL UNIQUE,
    refresh_hash VARCHAR(64) NOT NULL UNIQUE,
    roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    user_agent TEXT,
    ip_address VARCHAR(64),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id
    ON auth_sessions (user_id);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at
    ON auth_sessions (expires_at);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_active_user
    ON auth_sessions (user_id, expires_at)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS auth_sessions;
