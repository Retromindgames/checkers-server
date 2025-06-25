package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Default *zap.SugaredLogger

func init() {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel) // <-- set desired log level here
	//zap.DebugLevel   // show everything
	//zap.InfoLevel    // typical prod default
	//zap.WarnLevel    // only warnings and up
	//zap.ErrorLevel   // only errors and up
	//zap.FatalLevel   // suppress everything except fatal

	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // human-readable time

	rawLogger, _ := cfg.Build()
	Default = rawLogger.WithOptions(zap.AddCaller()).Sugar()
	Default.Info("logger initialized")
}
