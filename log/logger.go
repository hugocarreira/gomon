package log

import (
	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func InitLogger() {
	var err error

	cfg := zap.Config{
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

	logger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
}

func Info(m string, fields ...zap.Field) {
	c := color.GreenString(m)
	logger.Info(c, fields...)
}

func Warn(m string, fields ...zap.Field) {
	c := color.YellowString(m)
	logger.Warn(c, fields...)
}

func Error(m string, fields ...zap.Field) {
	c := color.RedString(m)
	logger.Error(c, fields...)
}

func Debug(m string, fields ...zap.Field) {
	c := color.CyanString(m)
	logger.Debug(c, fields...)
}

func Fatal(m string, fields ...zap.Field) {
	c := color.RedString(m)
	logger.Fatal(c, fields...)
}
