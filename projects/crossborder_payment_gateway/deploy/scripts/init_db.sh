#!/bin/bash
# PostgreSQL initialization script for cross-border payment gateway

set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create extensions
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS "pgcrypto";

    -- Create tables
    CREATE TABLE IF NOT EXISTS users (
        id TEXT PRIMARY KEY,
        username TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'merchant',
        email TEXT DEFAULT '',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS merchants (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        api_key TEXT NOT NULL,
        api_secret TEXT NOT NULL,
        status TEXT NOT NULL DEFAULT 'active',
        daily_limit BIGINT DEFAULT 100000000,
        monthly_limit BIGINT DEFAULT 1000000000,
        callback_url TEXT DEFAULT '',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS accounts (
        id TEXT PRIMARY KEY,
        merchant_id TEXT NOT NULL,
        currency TEXT NOT NULL,
        balance BIGINT DEFAULT 0,
        reserved_balance BIGINT DEFAULT 0,
        status TEXT NOT NULL DEFAULT 'active',
        daily_limit BIGINT DEFAULT 100000000,
        daily_used BIGINT DEFAULT 0,
        monthly_limit BIGINT DEFAULT 1000000000,
        monthly_used BIGINT DEFAULT 0,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS transactions (
        id TEXT PRIMARY KEY,
        merchant_id TEXT NOT NULL,
        payer_account_id TEXT NOT NULL,
        payee_account_id TEXT NOT NULL,
        source_currency TEXT NOT NULL,
        target_currency TEXT NOT NULL,
        source_amount BIGINT NOT NULL,
        target_amount BIGINT DEFAULT 0,
        exchange_rate DOUBLE PRECISION DEFAULT 0,
        fee BIGINT DEFAULT 0,
        status TEXT NOT NULL DEFAULT 'pending',
        description TEXT DEFAULT '',
        reference_id TEXT DEFAULT '',
        callback_url TEXT DEFAULT '',
        hash_chain_prev TEXT DEFAULT '',
        hash_chain_curr TEXT DEFAULT '',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS audit_logs (
        id BIGSERIAL PRIMARY KEY,
        user_id TEXT DEFAULT '',
        action TEXT NOT NULL,
        resource TEXT NOT NULL,
        resource_id TEXT DEFAULT '',
        ip_addr TEXT DEFAULT '',
        user_agent TEXT DEFAULT '',
        detail TEXT DEFAULT '',
        created_at TIMESTAMPTZ DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS exchange_rates (
        id BIGSERIAL PRIMARY KEY,
        source TEXT NOT NULL,
        target TEXT NOT NULL,
        rate DOUBLE PRECISION NOT NULL,
        bid DOUBLE PRECISION DEFAULT 0,
        ask DOUBLE PRECISION DEFAULT 0,
        source_name TEXT DEFAULT '',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        UNIQUE(source, target)
    );

    -- Create indexes
    CREATE INDEX IF NOT EXISTS idx_txn_status ON transactions(status);
    CREATE INDEX IF NOT EXISTS idx_txn_merchant ON transactions(merchant_id);
    CREATE INDEX IF NOT EXISTS idx_txn_created ON transactions(created_at DESC);
    CREATE INDEX IF NOT EXISTS idx_accounts_merchant ON accounts(merchant_id);
    CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
    CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at DESC);

    -- Insert default admin user (password: admin123, bcrypt hashed)
    INSERT INTO users (id, username, password_hash, role, email)
    VALUES ('usr-admin-001', 'admin', '\$2a\$10\$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin', 'admin@aspira.com')
    ON CONFLICT (username) DO NOTHING;

    -- Insert sample merchants
    INSERT INTO merchants (id, name, api_key, api_secret, status, daily_limit, monthly_limit)
    VALUES
        ('merchant-001', 'GlobalTrade Corp', 'ak-globaltrade-001', 'sk-secret-merchant1', 'active', 100000000, 1000000000),
        ('merchant-002', 'CrossBorderPay Ltd', 'ak-crossborder-002', 'sk-secret-merchant2', 'active', 50000000, 500000000)
    ON CONFLICT (id) DO NOTHING;

    -- Insert sample accounts
    INSERT INTO accounts (id, merchant_id, currency, balance, status)
    VALUES
        ('acct-001', 'merchant-001', 'USD', 100000000, 'active'),
        ('acct-002', 'merchant-001', 'CNY', 10000000000, 'active'),
        ('acct-003', 'merchant-002', 'USD', 50000000, 'active'),
        ('acct-004', 'merchant-002', 'CNY', 5000000000, 'active')
    ON CONFLICT (id) DO NOTHING;

    -- Insert exchange rates
    INSERT INTO exchange_rates (source, target, rate, bid, ask, source_name)
    VALUES
        ('USD', 'CNY', 7.2530, 7.2480, 7.2580, 'CFETS'),
        ('USD', 'EUR', 0.9215, 0.9200, 0.9230, 'ECB'),
        ('USD', 'JPY', 155.75, 155.50, 156.00, 'BOJ'),
        ('USD', 'GBP', 0.7910, 0.7895, 0.7925, 'BOE'),
        ('EUR', 'CNY', 7.8740, 7.8680, 7.8800, 'CFETS')
    ON CONFLICT (source, target) DO NOTHING;

    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO paygw;
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO paygw;
EOSQL

echo "PostgreSQL initialization complete."
