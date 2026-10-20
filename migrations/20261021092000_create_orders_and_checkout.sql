-- +goose Up
CREATE TYPE order_status AS ENUM (
    'pending_payment',
    'payment_failed',
    'paid',
    'processing',
    'completed',
    'cancelled'
);

CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    public_id UUID NOT NULL UNIQUE,
    buyer_id BIGINT NOT NULL REFERENCES users(id),
    status order_status NOT NULL,
    total_amount BIGINT NOT NULL CHECK (total_amount >= 0),
    payment_reference VARCHAR(128) NOT NULL UNIQUE,
    payment_expires_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_buyer_created_at
    ON orders (buyer_id, created_at);

CREATE INDEX IF NOT EXISTS idx_orders_payment_expiration
    ON orders (status, payment_expires_at);

CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id),
    seller_id BIGINT NOT NULL REFERENCES users(id),
    product_name VARCHAR(255) NOT NULL,
    unit_price_amount BIGINT NOT NULL CHECK (unit_price_amount >= 0),
    quantity INT NOT NULL CHECK (quantity > 0),
    subtotal_amount BIGINT NOT NULL CHECK (subtotal_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id
    ON order_items (order_id);

CREATE INDEX IF NOT EXISTS idx_order_items_seller_created_at
    ON order_items (seller_id, created_at);

CREATE INDEX IF NOT EXISTS idx_order_items_product_id
    ON order_items (product_id);

CREATE TABLE IF NOT EXISTS checkout_idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    buyer_id BIGINT NOT NULL REFERENCES users(id),
    idempotency_key VARCHAR(255) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    order_id BIGINT REFERENCES orders(id),
    response_status INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_checkout_idempotency_buyer_key UNIQUE (buyer_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_checkout_idempotency_order_id
    ON checkout_idempotency_keys (order_id);

-- +goose Down
DROP TABLE IF EXISTS checkout_idempotency_keys;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_status;
