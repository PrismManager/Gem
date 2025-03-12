package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInitLogger(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gem-logger-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize logger
	logFile := filepath.Join(tempDir, "test.log")
	err = InitLogger(logFile)
	assert.NoError(t, err)

	// Check if log file was created
	_, err = os.Stat(logFile)
	assert.NoError(t, err)

	// Write a log entry
	logrus.Info("Test log entry")

	// Check if log file contains the entry
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Test log entry")
}

func TestSetLogLevel(t *testing.T) {
	// Test different log levels
	testCases := []struct {
		level    string
		expected logrus.Level
	}{
		{"debug", logrus.DebugLevel},
		{"info", logrus.InfoLevel},
		{"warn", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"invalid", logrus.InfoLevel}, // Default to info for invalid levels
	}

	for _, tc := range testCases {
		SetLogLevel(tc.level)
		assert.Equal(t, tc.expected, logrus.GetLevel())
	}
}
