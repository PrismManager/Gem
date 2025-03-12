package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// InitLogger initializes the global logger
func InitLogger(logFile string) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Set up multi-writer for both file and stdout
	mw := io.MultiWriter(os.Stdout, file)
	logrus.SetOutput(mw)

	// Set log format
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set default log level
	logrus.SetLevel(logrus.InfoLevel)

	return nil
}

// SetLogLevel sets the log level based on a string
func SetLogLevel(level string) {
	switch level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}
