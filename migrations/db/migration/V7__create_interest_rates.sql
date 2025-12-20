-- Interest rates from Vietnamese banks
CREATE TABLE IF NOT EXISTS interest_rates (
    id SERIAL PRIMARY KEY,
    bank_code VARCHAR(20) NOT NULL,
    bank_name VARCHAR(100) NOT NULL,
    bank_logo VARCHAR(255),
    product_type VARCHAR(50) NOT NULL DEFAULT 'deposit', -- deposit, loan, mortgage
    term_months INT NOT NULL DEFAULT 0,
    term_label VARCHAR(50),
    rate DECIMAL(5,2) NOT NULL, -- e.g., 6.50 for 6.5%
    min_amount DECIMAL(15,2) DEFAULT 0,
    max_amount DECIMAL(15,2),
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',
    effective_date DATE NOT NULL,
    scraped_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX idx_interest_rates_bank_code ON interest_rates(bank_code);
CREATE INDEX idx_interest_rates_product_type ON interest_rates(product_type);
CREATE INDEX idx_interest_rates_term_months ON interest_rates(term_months);
CREATE INDEX idx_interest_rates_effective_date ON interest_rates(effective_date);

-- Composite index for common queries
CREATE INDEX idx_interest_rates_lookup ON interest_rates(product_type, term_months, bank_code);

-- Unique constraint for upsert operations
CREATE UNIQUE INDEX idx_interest_rates_unique ON interest_rates(bank_code, product_type, term_months);

-- Comments
COMMENT ON TABLE interest_rates IS 'Bank interest rates scraped from Vietnamese banks';
COMMENT ON COLUMN interest_rates.bank_code IS 'Short code for the bank (e.g., vcb, tcb, mb)';
COMMENT ON COLUMN interest_rates.product_type IS 'Type of product: deposit, loan, mortgage';
COMMENT ON COLUMN interest_rates.term_months IS 'Term duration in months (0 = no term/demand)';
COMMENT ON COLUMN interest_rates.rate IS 'Annual interest rate as percentage (e.g., 6.50 for 6.5%)';

