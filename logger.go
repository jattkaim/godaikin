package godaikin

import (
	"log/slog"
)

// Logger defines the logging interface for the library
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NoOpLogger is a logger that does nothing (silent by default)
type NoOpLogger struct{}

func (NoOpLogger) Debug(string, ...any) {}
func (NoOpLogger) Info(string, ...any)  {}
func (NoOpLogger) Warn(string, ...any)  {}
func (NoOpLogger) Error(string, ...any) {}

// SlogAdapter adapts slog.Logger to our Logger interface
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new SlogAdapter
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogAdapter{logger: logger}
}

func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *SlogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}
