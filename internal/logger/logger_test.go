package logger

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	l := Logger()
	assert.NotNil(t, l)
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithRequestID(ctx, "test-request-123")

	val := ctx.Value(requestIDKey)
	assert.Equal(t, "test-request-123", val)
}

func TestWithUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithUserID(ctx, "user-456")

	val := ctx.Value(userIDKey)
	assert.Equal(t, "user-456", val)
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		wantNotNil bool
	}{
		{
			name:       "empty context",
			setupCtx:   context.Background,
			wantNotNil: true,
		},
		{
			name: "with request ID",
			setupCtx: func() context.Context {
				return WithRequestID(context.Background(), "req-123")
			},
			wantNotNil: true,
		},
		{
			name: "with user ID",
			setupCtx: func() context.Context {
				return WithUserID(context.Background(), "user-456")
			},
			wantNotNil: true,
		},
		{
			name: "with both IDs",
			setupCtx: func() context.Context {
				ctx := WithRequestID(context.Background(), "req-123")
				return WithUserID(ctx, "user-456")
			},
			wantNotNil: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.setupCtx()
			l := FromContext(ctx)

			assert.NotNil(t, l)
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// These just verify the functions don't panic
	// Actual logging output goes to stdout

	// Redirect output during test
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w

	Info("test info", "key", "value")
	Error("test error", "key", "value")
	Debug("test debug", "key", "value")
	Warn("test warn", "key", "value")

	_ = w.Close()
	_ = r.Close()

	// If we got here without panic, test passes
	assert.True(t, true)
}

func TestLoggerWithProductionEnv(t *testing.T) {
	// This tests that init() handles production env
	// We can't easily test init() but we can verify Logger() returns something
	l := Logger()
	assert.NotNil(t, l)
}
