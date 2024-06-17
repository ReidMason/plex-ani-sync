package logger

import "github.com/charmbracelet/log"

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
}

type MockLogger struct{}

func (m MockLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Info(msg, keysAndValues...)
}

func (m MockLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Error(msg, keysAndValues...)
}

func (m MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Warn(msg, keysAndValues...)
}
