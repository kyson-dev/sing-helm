package logger_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyson/minibox/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_Setup(t *testing.T) {
	logger.ResetForTest() // Reset state

	// Test instance setup with Debug
	logger.Setup(logger.Config{
		Debug: true,
	})
	l := logger.Get()
	assert.NotNil(t, l, "logger instance should not be nil")

	// Test set level
	ctx := context.Background()
	assert.True(t, l.Enabled(ctx, slog.LevelDebug), "logger level should be debug")
}

func TestLogger_FileConfig(t *testing.T) {
	logger.ResetForTest() // Reset state

	// Create temp dir for log file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Setup logger with file path
	logger.Setup(logger.Config{
		Debug:    true,
		FilePath: logPath,
	})

	// Log something
	logger.Get().Info("test file log content")

	// Verify file exists
	assert.FileExists(t, logPath)

	// Verify content
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test file log content")
}
