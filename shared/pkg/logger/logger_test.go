package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/alokwritescode/digi-vault/shared/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ReturnsLogger(t *testing.T) {
	l := logger.New()
	assert.NotNil(t, l)
}

func TestLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter(&buf)

	l.Info("test message")

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err, "output should be valid JSON")
	assert.Equal(t, "test message", entry["msg"])
	assert.Equal(t, "info", entry["level"])
}

func TestLogger_WithContext_InjectsRequestID(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter(&buf)

	ctx := logger.ContextWithRequestID(context.Background(), "req-abc-123")
	l.WithContext(ctx).Info("handling request")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "req-abc-123", entry["request_id"], "request_id should be injected from context")
}

func TestLogger_WithContext_NoRequestID(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter(&buf)

	l.WithContext(context.Background()).Info("no request id")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	_, hasID := entry["request_id"]
	assert.False(t, hasID, "request_id should not appear when not set")
}

func TestContextWithRequestID_RoundTrip(t *testing.T) {
	ctx := logger.ContextWithRequestID(context.Background(), "test-id-999")
	got := logger.RequestIDFromContext(ctx)
	assert.Equal(t, "test-id-999", got)
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	got := logger.RequestIDFromContext(context.Background())
	assert.Empty(t, got)
}
