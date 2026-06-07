-- 010: Add default_currency to users table
-- Aspira Pay V2 — User's preferred display currency

ALTER TABLE users ADD COLUMN IF NOT EXISTS default_currency VARCHAR(8) NOT NULL DEFAULT 'USD';

-- Set admin user's default currency to USD
UPDATE users SET default_currency = 'USD' WHERE default_currency IS NULL OR default_currency = '';
