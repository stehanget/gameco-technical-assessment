-- +goose Up
CREATE TYPE stock_movement_type AS ENUM ('checkout', 'cancellation', 'manual_adjustment', 'restock');

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id BIGINT REFERENCES users(id),
    event_type VARCHAR(128) NOT NULL,
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address VARCHAR(64),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_aggregate
    ON audit_events (aggregate_type, aggregate_id, created_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_actor
    ON audit_events (actor_user_id, created_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_type
    ON audit_events (event_type, created_at);

CREATE TABLE IF NOT EXISTS stock_movements (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id),
    order_id BIGINT REFERENCES orders(id),
    order_item_id BIGINT REFERENCES order_items(id),
    actor_user_id BIGINT REFERENCES users(id),
    movement_type stock_movement_type NOT NULL,
    quantity_delta INT NOT NULL CHECK (quantity_delta <> 0),
    stock_before INT NOT NULL CHECK (stock_before >= 0),
    stock_after INT NOT NULL CHECK (stock_after >= 0),
    reason VARCHAR(255) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_created_at
    ON stock_movements (product_id, created_at);

CREATE INDEX IF NOT EXISTS idx_stock_movements_order_id
    ON stock_movements (order_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_order_item_id
    ON stock_movements (order_item_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_actor_created_at
    ON stock_movements (actor_user_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS audit_events;
DROP TYPE IF EXISTS stock_movement_type;
