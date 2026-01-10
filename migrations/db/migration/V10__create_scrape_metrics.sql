-- Scrape metrics table for tracking bank scraping operations
CREATE TABLE scrape_metrics (
    id SERIAL PRIMARY KEY,
    bank_code VARCHAR(20) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    rates_scraped INT DEFAULT 0,
    success BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    duration_ms INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for querying by bank code
CREATE INDEX idx_scrape_metrics_bank_code ON scrape_metrics(bank_code);

-- Index for querying recent metrics
CREATE INDEX idx_scrape_metrics_started_at ON scrape_metrics(started_at DESC);

-- Index for querying by success status
CREATE INDEX idx_scrape_metrics_success ON scrape_metrics(success);

-- Composite index for bank success rate queries
CREATE INDEX idx_scrape_metrics_bank_success ON scrape_metrics(bank_code, success, started_at DESC);

COMMENT ON TABLE scrape_metrics IS 'Tracks interest rate scraping operations for monitoring and debugging';
COMMENT ON COLUMN scrape_metrics.bank_code IS 'Bank identifier (e.g., vcb, tcb, mb)';
COMMENT ON COLUMN scrape_metrics.started_at IS 'When the scrape operation started';
COMMENT ON COLUMN scrape_metrics.completed_at IS 'When the scrape operation completed';
COMMENT ON COLUMN scrape_metrics.rates_scraped IS 'Number of interest rates successfully scraped';
COMMENT ON COLUMN scrape_metrics.success IS 'Whether the scrape operation was successful';
COMMENT ON COLUMN scrape_metrics.error_message IS 'Error message if the scrape failed';
COMMENT ON COLUMN scrape_metrics.duration_ms IS 'Duration of the scrape operation in milliseconds';
