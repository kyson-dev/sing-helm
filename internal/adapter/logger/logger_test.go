package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	// Test instance
	logger.Setup(true)
	l := logger.Get()
	assert.NotNil(t, l, "logger instance should not be nil")

	// Test set level
	ctx := context.Background()
	assert.True(t, l.Enabled(ctx, slog.LevelDebug), "logger level should be debug")

}
func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	l := slog.New(handler)

	l.Info("test info")

	output := buf.String()
	assert.Contains(t, output, "test info")
}
