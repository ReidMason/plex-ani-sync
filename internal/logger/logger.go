package logger

import "github.com/charmbracelet/log"

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

type MockLogger struct{}

func (m MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	log.Debug(msg, keysAndValues...)
}

func (m MockLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Info(msg, keysAndValues...)
}

func (m MockLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Error(msg, keysAndValues...)
}

func (m MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Warn(msg, keysAndValues...)
}
