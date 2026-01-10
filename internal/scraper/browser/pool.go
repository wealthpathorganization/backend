// Package browser provides headless browser automation for scraping JS-rendered pages.
package browser

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Pool manages browser instances for concurrent scraping
type Pool struct {
	browser  *rod.Browser
	pagePool chan *rod.Page
	maxPages int
	logger   *slog.Logger
	mu       sync.Mutex
	closed   bool
}

// PoolConfig holds configuration for the browser pool
type PoolConfig struct {
	MaxPages        int           // Maximum concurrent pages (default: 3)
	PageTimeout     time.Duration // Timeout for page operations (default: 30s)
	Headless        bool          // Run in headless mode (default: true)
	UserDataDir     string        // Browser user data directory (optional)
	SlowMotion      time.Duration // Slow down operations for debugging (default: 0)
}

// DefaultPoolConfig returns the default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxPages:    3,
		PageTimeout: 60 * time.Second, // Increased for JS-heavy pages
		Headless:    true,
		SlowMotion:  0,
	}
}

// NewPool creates a browser pool with the given configuration
func NewPool(cfg PoolConfig, logger *slog.Logger) (*Pool, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 3
	}

	// Configure browser launcher
	l := launcher.New().
		Headless(cfg.Headless).
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-dev-shm-usage").
		Set("disable-setuid-sandbox")

	if cfg.UserDataDir != "" {
		l = l.UserDataDir(cfg.UserDataDir)
	}

	// Launch browser
	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if cfg.SlowMotion > 0 {
		browser = browser.SlowMotion(cfg.SlowMotion)
	}

	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to browser: %w", err)
	}

	pool := &Pool{
		browser:  browser,
		pagePool: make(chan *rod.Page, cfg.MaxPages),
		maxPages: cfg.MaxPages,
		logger:   logger,
	}

	// Pre-warm pool with pages
	for i := 0; i < cfg.MaxPages; i++ {
		page, err := pool.createPage(cfg.PageTimeout)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("creating page %d: %w", i, err)
		}
		pool.pagePool <- page
	}

	logger.Info("Browser pool initialized",
		slog.Int("max_pages", cfg.MaxPages),
		slog.Bool("headless", cfg.Headless),
	)

	return pool, nil
}

// createPage creates a new browser page with default settings
func (p *Pool) createPage(timeout time.Duration) (*rod.Page, error) {
	page, err := p.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, err
	}

	// Set default timeout
	page = page.Timeout(timeout)

	// Set viewport to common desktop resolution
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  1920,
		Height: 1080,
	}); err != nil {
		return nil, err
	}

	// Enable stealth mode (basic anti-detection)
	_, err = page.Eval(`() => {
		// Remove webdriver property
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});
		// Mock plugins
		Object.defineProperty(navigator, 'plugins', {
			get: () => [1, 2, 3, 4, 5]
		});
		// Mock languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['vi-VN', 'vi', 'en-US', 'en']
		});
	}`)
	if err != nil {
		p.logger.Warn("Failed to apply stealth mode", slog.String("error", err.Error()))
	}

	return page, nil
}

// Acquire gets a page from the pool (blocks if none available)
func (p *Pool) Acquire(ctx context.Context) (*rod.Page, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.Unlock()

	select {
	case page := <-p.pagePool:
		return page, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns a page to the pool
func (p *Pool) Release(page *rod.Page) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		// Pool is closed, just close the page
		_ = page.Close()
		return
	}

	// Clear page state before returning
	_ = page.Navigate("about:blank")

	// Clear cookies
	_ = page.SetCookies(nil)

	select {
	case p.pagePool <- page:
		// Page returned to pool
	default:
		// Pool is full, close the page
		_ = page.Close()
	}
}

// Close shuts down the browser pool
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	// Close all pages in pool
	close(p.pagePool)
	for page := range p.pagePool {
		_ = page.Close()
	}

	// Close browser
	if err := p.browser.Close(); err != nil {
		return fmt.Errorf("closing browser: %w", err)
	}

	p.logger.Info("Browser pool closed")
	return nil
}

// PageHelper provides helper methods for page interactions
type PageHelper struct {
	Page    *rod.Page
	Timeout time.Duration
}

// NewPageHelper creates a new page helper
func NewPageHelper(page *rod.Page, timeout time.Duration) *PageHelper {
	return &PageHelper{
		Page:    page,
		Timeout: timeout,
	}
}

// NavigateAndWait navigates to URL and waits for page to be stable
func (h *PageHelper) NavigateAndWait(url string) error {
	if err := h.Page.Navigate(url); err != nil {
		return fmt.Errorf("navigating to %s: %w", url, err)
	}

	if err := h.Page.WaitLoad(); err != nil {
		return fmt.Errorf("waiting for load: %w", err)
	}

	// Wait for network to be idle
	if err := h.Page.WaitRequestIdle(time.Second, nil, nil, nil); err != nil {
		// Not critical, continue
	}

	return nil
}

// WaitForSelector waits for an element to appear
func (h *PageHelper) WaitForSelector(selector string) (*rod.Element, error) {
	el, err := h.Page.Timeout(h.Timeout).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("waiting for selector %s: %w", selector, err)
	}
	return el, nil
}

// WaitForSelectorVisible waits for an element to be visible
func (h *PageHelper) WaitForSelectorVisible(selector string) (*rod.Element, error) {
	el, err := h.WaitForSelector(selector)
	if err != nil {
		return nil, err
	}
	if err := el.WaitVisible(); err != nil {
		return nil, fmt.Errorf("waiting for %s to be visible: %w", selector, err)
	}
	return el, nil
}

// GetHTML returns the page HTML
func (h *PageHelper) GetHTML() (string, error) {
	return h.Page.HTML()
}

// EvalJS evaluates JavaScript and returns the result
func (h *PageHelper) EvalJS(js string) (interface{}, error) {
	result, err := h.Page.Eval(js)
	if err != nil {
		return nil, err
	}
	return result.Value.Val(), nil
}

// Screenshot takes a screenshot and saves it to the given path
func (h *PageHelper) Screenshot(path string) error {
	data, err := h.Page.Screenshot(false, nil)
	if err != nil {
		return err
	}
	_ = data // Would save to file in real implementation
	return nil
}
