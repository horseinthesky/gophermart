package service

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logLevelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

var logEncoderCreatorMap = map[string]func(cfg zapcore.EncoderConfig) zapcore.Encoder{
	"json":   zapcore.NewJSONEncoder,
	"printf": zapcore.NewConsoleEncoder,
}

func initLogger(logLevel, logFormat string) (*zap.SugaredLogger, error) {
	defaultLogLevel, ok := logLevelMap[logLevel]
	if !ok {
		return nil, fmt.Errorf(`failed to init logger: level "%s" is not supported`, logLevel)
	}

	encoderCreator, ok := logEncoderCreatorMap[logFormat]
	if !ok {
		return nil, fmt.Errorf(`failed to init logger: format "%s" is not supported`, logFormat)
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.RFC3339TimeEncoder

	core := zapcore.NewTee(
		zapcore.NewCore(encoderCreator(config), zapcore.AddSync(os.Stdout), defaultLogLevel),
	)

	return zap.New(core, zap.AddCaller()).Sugar(), nil
}
