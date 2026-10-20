-- +goose Up
CREATE TABLE IF NOT EXISTS user_login_security_states (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    failed_attempt_count INT NOT NULL DEFAULT 0 CHECK (failed_attempt_count >= 0),
    first_failed_at TIMESTAMPTZ,
    last_failed_at TIMESTAMPTZ,
    locked_until TIMESTAMPTZ,
    requires_manual_unlock BOOLEAN NOT NULL DEFAULT FALSE,
    last_lock_reason VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS user_login_security_states;
