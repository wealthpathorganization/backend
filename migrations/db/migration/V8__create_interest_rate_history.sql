-- Historical interest rate tracking for charts and analysis
CREATE TABLE IF NOT EXISTS interest_rate_history (
    id SERIAL PRIMARY KEY,
    bank_code VARCHAR(20) NOT NULL,
    product_type VARCHAR(50) NOT NULL,
    term_months INT NOT NULL,
    rate DECIMAL(5,2) NOT NULL,
    recorded_date DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient historical queries
CREATE INDEX idx_rate_history_bank_date ON interest_rate_history(bank_code, recorded_date);
CREATE INDEX idx_rate_history_lookup ON interest_rate_history(product_type, term_months, recorded_date);

-- Unique constraint to prevent duplicate entries for same bank/product/term/date
CREATE UNIQUE INDEX idx_rate_history_unique ON interest_rate_history(bank_code, product_type, term_months, recorded_date);

-- Function to record rate history when rates change
CREATE OR REPLACE FUNCTION record_rate_history()
RETURNS TRIGGER AS $$
BEGIN
    -- Only record if rate actually changed or it's a new rate
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND OLD.rate IS DISTINCT FROM NEW.rate) THEN
        INSERT INTO interest_rate_history (bank_code, product_type, term_months, rate, recorded_date)
        VALUES (NEW.bank_code, NEW.product_type, NEW.term_months, NEW.rate, CURRENT_DATE)
        ON CONFLICT (bank_code, product_type, term_months, recorded_date) 
        DO UPDATE SET rate = EXCLUDED.rate;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically record history on rate changes
DROP TRIGGER IF EXISTS trg_record_rate_history ON interest_rates;
CREATE TRIGGER trg_record_rate_history
    AFTER INSERT OR UPDATE ON interest_rates
    FOR EACH ROW
    EXECUTE FUNCTION record_rate_history();

-- Comments
COMMENT ON TABLE interest_rate_history IS 'Historical interest rate data for trend analysis';
COMMENT ON COLUMN interest_rate_history.recorded_date IS 'Date when this rate was recorded (one entry per day max)';

