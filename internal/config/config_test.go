package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment to test defaults
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("DATABASE_URL")

	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Contains(t, cfg.DatabaseURL, "postgres://")
	assert.NotEmpty(t, cfg.JWTSecret)
	assert.Contains(t, cfg.AllowedOrigins, "http://localhost:3000")
}

func TestLoad_WithEnvVars(t *testing.T) {
	// Set test environment variables
	t.Setenv("PORT", "9090")
	t.Setenv("ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://test:5432/testdb")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ALLOWED_ORIGINS", "http://example.com,http://test.com")
	t.Setenv("GOOGLE_CLIENT_ID", "google-id")
	t.Setenv("OPENAI_API_KEY", "openai-key")
	t.Setenv("ENABLE_AI_CHAT", "false")

	cfg := Load()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "postgres://test:5432/testdb", cfg.DatabaseURL)
	assert.Equal(t, "test-secret", cfg.JWTSecret)
	assert.Len(t, cfg.AllowedOrigins, 2)
	assert.Equal(t, "google-id", cfg.GoogleClientID)
	assert.Equal(t, "openai-key", cfg.OpenAIAPIKey)
	assert.False(t, cfg.EnableAIChat)
}

func TestConfig_IsDevelopment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		env      string
		expected bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.env, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{Env: tt.env}
			assert.Equal(t, tt.expected, cfg.IsDevelopment())
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		env      string
		expected bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.env, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{Env: tt.env}
			assert.Equal(t, tt.expected, cfg.IsProduction())
		})
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_VAR", "test_value")

	assert.Equal(t, "test_value", getEnv("TEST_VAR", "default"))
	assert.Equal(t, "default", getEnv("NON_EXISTENT_VAR", "default"))
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		defaultValue bool
		expected     bool
	}{
		{"true value", "true", true, false, true},
		{"false value", "false", true, true, false},
		{"1 value", "1", true, false, true},
		{"0 value", "0", true, true, false},
		{"invalid value uses default", "invalid", true, true, true},
		{"unset uses default true", "", false, true, true},
		{"unset uses default false", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv("TEST_BOOL", tt.envValue)
			} else {
				_ = os.Unsetenv("TEST_BOOL")
			}
			assert.Equal(t, tt.expected, getBoolEnv("TEST_BOOL", tt.defaultValue))
		})
	}
}
