CREATE TABLE wallet_table (
    wallet_id BIGSERIAL PRIMARY KEY,
    balance   NUMERIC(19,4) NOT NULL DEFAULT 0
);
