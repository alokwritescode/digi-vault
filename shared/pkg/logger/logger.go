package logger

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// requestIDHook injects request_id from context into every log entry.
type requestIDHook struct{}

func (h *requestIDHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *requestIDHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil
	}
	if id, ok := entry.Context.Value(requestIDKey).(string); ok && id != "" {
		entry.Data["request_id"] = id
	}
	return nil
}

// New returns a production logger writing JSON to stdout.
func New() *logrus.Logger {
	return NewWithWriter(os.Stdout)
}

// NewWithWriter returns a logger writing JSON to the given writer (useful in tests).
func NewWithWriter(w io.Writer) *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetOutput(w)
	l.SetLevel(logrus.InfoLevel)
	l.AddHook(&requestIDHook{})
	return l
}

// ContextWithRequestID stores a request ID in the context.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
