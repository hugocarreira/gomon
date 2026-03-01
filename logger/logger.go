// Package logger provides colored logging functionality for gomon.
package logger

import (
	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ILogger defines the interface for logging operations.
type ILogger interface {
	Info(m string, fields ...zap.Field)
	Warn(m string, fields ...zap.Field)
	Error(m string, fields ...zap.Field)
	Debug(m string, fields ...zap.Field)
	Fatal(m string, fields ...zap.Field)
	Building(m string)
	Running(m string)
	BuildSuccess(m string)
	BuildError(m string)
}

// Logger implements the ILogger interface using zap and fatih/color.
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates a new Logger instance with console output.
func NewLogger() ILogger {
	logConfig := zap.Config{
		Encoding:         "console",
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",
			LevelKey:   "",
			TimeKey:    "",
			CallerKey:  "",
		},
	}

	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{logger: logger}
}

// Info logs a message at info level with green color.
func (l *Logger) Info(m string, fields ...zap.Field) {
	c := color.GreenString(m)
	l.logger.Info(c, fields...)
}

// Warn logs a message at warn level with yellow color.
func (l *Logger) Warn(m string, fields ...zap.Field) {
	c := color.YellowString(m)
	l.logger.Warn(c, fields...)
}

// Error logs a message at error level with red color.
func (l *Logger) Error(m string, fields ...zap.Field) {
	c := color.RedString(m)
	l.logger.Error(c, fields...)
}

// Debug logs a message at debug level with cyan color.
func (l *Logger) Debug(m string, fields ...zap.Field) {
	c := color.CyanString(m)
	l.logger.Debug(c, fields...)
}

// Fatal logs a message at fatal level and exits.
func (l *Logger) Fatal(m string, fields ...zap.Field) {
	c := color.RedString(m)
	l.logger.Fatal(c, fields...)
}

// Building logs a build message with blue color.
func (l *Logger) Building(m string) {
	msg := color.BlueString("[BUILD]") + " " + m
	l.logger.Info(msg)
}

// Running logs a running message with green color.
func (l *Logger) Running(m string) {
	msg := color.GreenString("[RUN]") + " " + m
	l.logger.Info(msg)
}

// BuildSuccess logs a success message with green color.
func (l *Logger) BuildSuccess(m string) {
	msg := color.GreenString("[OK]") + " " + m
	l.logger.Info(msg)
}

// BuildError logs an error message with red color.
func (l *Logger) BuildError(m string) {
	msg := color.RedString("[ERROR]") + " " + m
	l.logger.Error(msg)
}
