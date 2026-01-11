-- Add rollover support to budgets
ALTER TABLE budgets ADD COLUMN IF NOT EXISTS enable_rollover BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE budgets ADD COLUMN IF NOT EXISTS max_rollover_amount NUMERIC(15,2);
ALTER TABLE budgets ADD COLUMN IF NOT EXISTS rollover_amount NUMERIC(15,2) NOT NULL DEFAULT 0;

-- Create budget_rollovers table to track rollover history
CREATE TABLE IF NOT EXISTS budget_rollovers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    from_period_start DATE NOT NULL,
    from_period_end DATE NOT NULL,
    to_period_start DATE NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(budget_id, from_period_start)
);

-- Index for querying rollovers by budget
CREATE INDEX IF NOT EXISTS idx_budget_rollovers_budget_id ON budget_rollovers(budget_id);

-- Comment on columns
COMMENT ON COLUMN budgets.enable_rollover IS 'Whether to carry over unspent budget to next period';
COMMENT ON COLUMN budgets.max_rollover_amount IS 'Maximum amount that can be rolled over (NULL = unlimited)';
COMMENT ON COLUMN budgets.rollover_amount IS 'Current rollover amount from previous period';
