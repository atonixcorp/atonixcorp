package utils

import (
	"log"
	"os"
)

// Logger is a simple wrapper around the standard Go logger.
type Logger struct {
	*log.Logger
}

// NewLogger creates a new logger with a prefix.
func NewLogger(prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lmicroseconds),
	}
}

// Info logs an informational message.
func (l *Logger) Info(msg string) {
	l.Println("[INFO]", msg)
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.Println("[ERROR]", msg)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	l.Println("[DEBUG]", msg)
}
