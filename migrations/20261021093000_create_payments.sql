-- +goose Up
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed', 'expired');

CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    public_id UUID NOT NULL UNIQUE,
    order_id BIGINT NOT NULL UNIQUE REFERENCES orders(id),
    payment_reference VARCHAR(128) NOT NULL UNIQUE,
    provider VARCHAR(64) NOT NULL,
    status payment_status NOT NULL,
    amount BIGINT NOT NULL CHECK (amount >= 0),
    expires_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_expiration
    ON payments (status, expires_at);

CREATE TABLE IF NOT EXISTS payment_webhook_events (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(64) NOT NULL,
    event_id VARCHAR(128) NOT NULL,
    payment_reference VARCHAR(128) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload_hash VARCHAR(64) NOT NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payment_webhook_events_provider_event UNIQUE (provider, event_id)
);

CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_payment_reference
    ON payment_webhook_events (payment_reference);

CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_processed_at
    ON payment_webhook_events (processed_at);

-- +goose Down
DROP TABLE IF EXISTS payment_webhook_events;
DROP TABLE IF EXISTS payments;
DROP TYPE IF EXISTS payment_status;
