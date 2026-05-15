CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    balance NUMERIC(18,2) NOT NULL CHECK (balance >= 0)
);

CREATE TABLE IF NOT EXISTS balance_debits (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
    balance_before NUMERIC(18,2) NOT NULL CHECK (balance_before >= 0),
    balance_after NUMERIC(18,2) NOT NULL CHECK (balance_after >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS balance_debits_user_created_idx
    ON balance_debits(user_id, created_at DESC);

INSERT INTO users (id, balance)
VALUES (1, 1000.00)
ON CONFLICT (id) DO NOTHING;
