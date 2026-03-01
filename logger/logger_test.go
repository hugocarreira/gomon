package logger

import "testing"

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
