package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// CookieConfig holds settings for HTTP cookies (refresh tokens).
type CookieConfig struct {
	Domain   string // Cookie domain (empty = current domain)
	Secure   bool   // Require HTTPS
	SameSite string // "Strict", "Lax", or "None"
	Path     string // Cookie path
}

type Config struct {
	// Server
	Port string
	Env  string // "development", "production"

	// Database
	DatabaseURL string

	// Auth
	JWTSecret string

	// Cookie settings for refresh tokens
	Cookie CookieConfig

	// CORS
	AllowedOrigins []string
	FrontendURL    string

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string

	FacebookAppID       string
	FacebookAppSecret   string
	FacebookRedirectURI string

	// AI
	OpenAIAPIKey string

	// Features
	EnableAIChat bool

	// Scraper
	ScraperEnabled  bool
	ScraperSchedule string        // Cron expression (e.g., "0 * * * *" for hourly)
	ScraperTimeout  time.Duration // Timeout for complete scrape cycle

	// Web Push Notifications
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string // mailto:email or URL
}

func Load() *Config {
	env := getEnv("ENV", "development")
	isProduction := env == "production"

	return &Config{
		// Server
		Port: getEnv("PORT", "8080"),
		Env:  env,

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/wealthpath?sslmode=disable"),

		// Auth
		JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-in-production"),

		// Cookie settings for refresh tokens
		Cookie: CookieConfig{
			Domain:   getEnv("COOKIE_DOMAIN", ""),                         // Empty = current domain
			Secure:   getBoolEnv("COOKIE_SECURE", isProduction),           // Require HTTPS in production
			SameSite: getEnv("COOKIE_SAME_SITE", "Strict"),                // Strict for CSRF protection
			Path:     getEnv("COOKIE_PATH", "/"),                          // Available on all paths
		},

		// CORS
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		FrontendURL:    getEnv("FRONTEND_URL", "http://localhost:3000"),

		// OAuth
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURI:  os.Getenv("GOOGLE_REDIRECT_URI"),

		FacebookAppID:       os.Getenv("FACEBOOK_APP_ID"),
		FacebookAppSecret:   os.Getenv("FACEBOOK_APP_SECRET"),
		FacebookRedirectURI: os.Getenv("FACEBOOK_REDIRECT_URI"),

		// AI
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),

		// Features
		EnableAIChat: getBoolEnv("ENABLE_AI_CHAT", true),

		// Scraper
		ScraperEnabled:  getBoolEnv("SCRAPER_ENABLED", true),
		ScraperSchedule: getEnv("SCRAPER_SCHEDULE", "0 * * * *"), // Default: hourly at minute 0
		ScraperTimeout:  getDurationEnv("SCRAPER_TIMEOUT", 5*time.Minute),

		// Web Push Notifications
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:    getEnv("VAPID_SUBJECT", "mailto:notifications@wealthpath.app"),
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
