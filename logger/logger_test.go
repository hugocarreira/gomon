package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLoggerMethods(t *testing.T) {
	l := NewLogger()

	l.Info("info")
	l.Warn("warn")
	l.Error("error")
	l.Debug("debug")
	l.Building("building")
	l.Running("running")
	l.BuildSuccess("ok")
	l.BuildError("failed")
}

func TestNewLoggerWithLevel(t *testing.T) {
	tests := []struct {
		level string
		want  zapcore.Level
	}{
		{level: "debug", want: zap.DebugLevel},
		{level: "info", want: zap.InfoLevel},
		{level: "warn", want: zap.WarnLevel},
		{level: "error", want: zap.ErrorLevel},
		{level: " INFO ", want: zap.InfoLevel},
		{level: "unknown", want: zap.DebugLevel},
	}
	for _, test := range tests {
		logger := NewLoggerWithLevel(test.level).(*Logger)
		if !logger.logger.Core().Enabled(test.want) {
			t.Fatalf("expected level %q to enable %s", test.level, test.want)
		}
	}
}
