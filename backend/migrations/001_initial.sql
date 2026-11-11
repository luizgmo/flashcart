-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash BYTEA NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    expires_at TIMESTAMPTZ NOT NULL,
    CHECK (expires_at > created_at)
);

CREATE TABLE products (
    id UUID PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    image_path TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents BETWEEN 1 AND 100000000),
    initial_stock INTEGER NOT NULL CHECK (initial_stock >= 0),
    stock_available INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CHECK (stock_available BETWEEN 0 AND initial_stock)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE RESTRICT,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity BETWEEN 1 AND 5),
    unit_price_cents BIGINT NOT NULL CHECK (unit_price_cents > 0),
    total_cents BIGINT GENERATED ALWAYS AS (unit_price_cents * quantity) STORED,
    status TEXT NOT NULL CHECK (status IN ('PENDING_PAYMENT','PAID','EXPIRED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    paid_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    CHECK (expires_at > created_at),
    CHECK (
      (status = 'PENDING_PAYMENT' AND paid_at IS NULL AND expired_at IS NULL) OR
      (status = 'PAID' AND paid_at IS NOT NULL AND expired_at IS NULL) OR
      (status = 'EXPIRED' AND expired_at IS NOT NULL AND paid_at IS NULL)
    )
);
CREATE INDEX orders_session_created_idx ON orders(session_id, created_at DESC, id DESC);
CREATE INDEX orders_expiry_idx ON orders(expires_at, id) WHERE status = 'PENDING_PAYMENT';

CREATE TABLE idempotency_keys (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE RESTRICT,
    key UUID NOT NULL,
    request_hash BYTEA NOT NULL CHECK (octet_length(request_hash) = 32),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (session_id, key)
);

CREATE TABLE order_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    type TEXT NOT NULL CHECK (type IN ('ORDER_CREATED','ORDER_PAID','ORDER_EXPIRED')),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (order_id, type)
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    type TEXT NOT NULL CHECK (type IN ('ORDER_PAID','ORDER_EXPIRED')),
    schema_version INTEGER NOT NULL DEFAULT 1 CHECK (schema_version = 1),
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    published_at TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    available_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    lease_token UUID,
    lease_until TIMESTAMPTZ,
    last_error TEXT CHECK (last_error IS NULL OR length(last_error) <= 500),
    UNIQUE (order_id, type)
);
CREATE INDEX outbox_pending_idx ON outbox_events(available_at, created_at) WHERE published_at IS NULL;

CREATE TABLE processed_events (
    event_id UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL UNIQUE REFERENCES processed_events(event_id) ON DELETE RESTRICT,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('ORDER_PAID','ORDER_EXPIRED')),
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX notifications_order_idx ON notifications(order_id, created_at);

CREATE TABLE quota_buckets (
    scope TEXT NOT NULL,
    subject_hash TEXT NOT NULL,
    action TEXT NOT NULL,
    bucket_start TIMESTAMPTZ NOT NULL,
    count INTEGER NOT NULL CHECK (count >= 0),
    PRIMARY KEY (scope, subject_hash, action, bucket_start)
);
CREATE INDEX quota_buckets_age_idx ON quota_buckets(bucket_start);

-- +goose Down
DROP TABLE IF EXISTS quota_buckets, notifications, processed_events, outbox_events,
  order_events, idempotency_keys, orders, products, sessions CASCADE;
DROP EXTENSION IF EXISTS pgcrypto;

