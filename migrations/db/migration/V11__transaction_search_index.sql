-- Enable pg_trgm extension for fuzzy text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add GIN index for efficient text search on transaction descriptions
CREATE INDEX IF NOT EXISTS idx_transactions_description_trgm
ON transactions USING GIN (description gin_trgm_ops);

-- Add index for amount range queries
CREATE INDEX IF NOT EXISTS idx_transactions_amount
ON transactions (amount);

-- Add composite index for common filter combinations
CREATE INDEX IF NOT EXISTS idx_transactions_user_date_type
ON transactions (user_id, date DESC, type);
