-- Initial schema migration
CREATE TABLE IF NOT EXISTS test_table (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);