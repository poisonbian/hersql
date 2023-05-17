package log

import (
	"os"

	"github.com/poisonbian/hersql/config"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GetLogger(conf *config.Log) *zap.SugaredLogger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	cores := make([]zapcore.Core, 1)
	cores[0] = zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.DebugLevel)

	if conf.InfoLogFilename != "" {
		lowPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
			return lev < zap.ErrorLevel && lev >= zap.DebugLevel
		})
		infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   conf.InfoLogFilename,
			MaxSize:    2,
			MaxBackups: 100,
			MaxAge:     30,
			Compress:   false,
		})
		infoFileCore := zapcore.NewCore(encoder, infoFileWriteSyncer, lowPriority)
		cores = append(cores, infoFileCore)
	}

	if conf.InfoLogFilename != "" {
		highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
			return lev >= zap.ErrorLevel
		})
		errorFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   conf.InfoLogFilename,
			MaxSize:    1,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   false,
		})
		errorFileCore := zapcore.NewCore(encoder, errorFileWriteSyncer, highPriority)
		cores = append(cores, errorFileCore)
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller()).Sugar()
}
