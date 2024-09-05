package log

import (
	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ILogger interface {
	Info(m string, fields ...zap.Field)
	Warn(m string, fields ...zap.Field)
	Error(m string, fields ...zap.Field)
	Debug(m string, fields ...zap.Field)
	Fatal(m string, fields ...zap.Field)
}

type Logger struct {
	logger *zap.Logger
}

func NewLogger() ILogger {
	var err error

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

func (l *Logger) Info(m string, fields ...zap.Field) {
	c := color.GreenString(m)
	l.logger.Info(c, fields...)
}

func (l *Logger) Warn(m string, fields ...zap.Field) {
	c := color.YellowString(m)
	l.logger.Warn(c, fields...)
}

func (l *Logger) Error(m string, fields ...zap.Field) {
	c := color.RedString(m)
	l.logger.Error(c, fields...)
}

func (l *Logger) Debug(m string, fields ...zap.Field) {
	c := color.CyanString(m)
	l.logger.Debug(c, fields...)
}

func (l *Logger) Fatal(m string, fields ...zap.Field) {
	c := color.RedString(m)
	l.logger.Fatal(c, fields...)
}
