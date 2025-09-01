package logging

import (
	"dice-sorensen-similarity-search/internal/config"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"moul.io/zapgorm2"
	"os"
	"runtime"
	"strings"
)

type Logger interface {
	LogErrorf(keyVal []any, format string, args ...any)
	LogError(keyVal []any, args ...any)
	LogWarnf(keyVal []any, format string, args ...any)
	LogWarn(keyVal []any, args ...any)
	LogInfof(keyVal []any, format string, args ...any)
	LogInfo(keyVal []any, args ...any)
	LogDebugf(keyVal []any, format string, args ...any)
	LogDebug(keyVal []any, args ...any)
}

type DefaultLogger struct {
	Logger *zap.SugaredLogger
}

// ensure DefaultLogger implements Logger
var _ Logger = &DefaultLogger{}

type NullLogger struct{}

// ensure NullLogger implements Logger
var _ Logger = &NullLogger{}

func InitLogging(c *config.Configuration) *DefaultLogger {
	var core zapcore.Core

	consoleEncoderCfg := zap.NewProductionEncoderConfig()
	consoleEncoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	consoleEncoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleWriteSyncer := zapcore.Lock(os.Stderr)

	fileEncoderCfg := zap.NewProductionEncoderConfig()
	fileEncoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoderCfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	var fileWriteSyncer zapcore.WriteSyncer
	if len(c.Logging.File) > 0 {
		fileWriteSyncer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   c.Logging.File,
			MaxSize:    c.Logging.MaxSize, // megabytes
			MaxBackups: c.Logging.MaxBackups,
			MaxAge:     c.Logging.MaxAge, // days
		})
		// if logfile is defined: log errors to console and configured log level to file
		core = zapcore.NewTee(
			zapcore.NewCore(
				zapcore.NewConsoleEncoder(consoleEncoderCfg),
				consoleWriteSyncer,
				c.Logging.ConsoleLogLevel,
			),
			zapcore.NewCore(
				zapcore.NewJSONEncoder(fileEncoderCfg),
				fileWriteSyncer,
				c.Logging.Level,
			),
		)
	} else {
		// log configured log level to console
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderCfg),
			consoleWriteSyncer,
			c.Logging.Level,
		)
	}

	zapLogger := zap.New(core)
	zap.ReplaceGlobals(zapLogger)
	zapSugaredLogger := zapLogger.Sugar()

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	return &DefaultLogger{Logger: zapSugaredLogger}
}

func InitGinLogger(c *config.Configuration) *zap.Logger {
	fileEncoderCfg := zap.NewProductionEncoderConfig()
	fileEncoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoderCfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	ginW := zapcore.AddSync(&lumberjack.Logger{
		Filename:   c.Logging.HttpAccessFile,
		MaxSize:    c.Logging.MaxSize, // megabytes
		MaxBackups: c.Logging.MaxBackups,
		MaxAge:     c.Logging.MaxAge, // days
	})
	ginCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(fileEncoderCfg),
		ginW,
		c.Logging.Level,
	)

	ginLogger := zap.New(ginCore)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	return ginLogger
}

func InitGormLogger(c *config.Configuration) *zapgorm2.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	// DB LOGGER
	gormW := zapcore.AddSync(&lumberjack.Logger{
		Filename:   c.Logging.DbLogFile,
		MaxSize:    c.Logging.MaxSize, // megabytes
		MaxBackups: c.Logging.MaxBackups,
		MaxAge:     c.Logging.MaxAge, // days
	})
	gormCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		gormW,
		c.Logging.Level,
	)
	zapGormLogger := zap.New(gormCore)

	gormLogger := zapgorm2.New(zapGormLogger)
	gormLogger.LogLevel = 4
	gormLogger.SetAsDefault()

	return &gormLogger
}

func (d DefaultLogger) RecoverPanic(description string) {
	if err := recover(); err != nil {
		d.LogError(nil, fmt.Sprintf("!!PANIC OCCURED!!: %v: %v\n%v", description, err, IdentifyPanic()))
	}
}

func IdentifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}

func (d DefaultLogger) LogErrorf(keyVal []any, format string, args ...any) {
	d.Logger.Errorw(fmt.Sprintf(format, args...), keyVal...)
}
func (d DefaultLogger) LogError(keyVal []any, args ...any) {
	d.Logger.Errorw(fmt.Sprint(args...), keyVal...)
}

func (d DefaultLogger) LogWarnf(keyVal []any, format string, args ...any) {
	d.Logger.Warnw(fmt.Sprintf(format, args...), keyVal...)
}
func (d DefaultLogger) LogWarn(keyVal []any, args ...any) {
	d.Logger.Warnw(fmt.Sprint(args...), keyVal...)
}

func (d DefaultLogger) LogInfof(keyVal []any, format string, args ...any) {
	d.Logger.Infow(fmt.Sprintf(format, args...), keyVal...)
}
func (d DefaultLogger) LogInfo(keyVal []any, args ...any) {
	d.Logger.Infow(fmt.Sprint(args...), keyVal...)
}

func (d DefaultLogger) LogDebugf(keyVal []any, format string, args ...any) {
	d.Logger.Debugw(fmt.Sprintf(format, args...), keyVal...)
}

func (d DefaultLogger) LogDebug(keyVal []any, args ...any) {
	d.Logger.Debugw(fmt.Sprint(args...), keyVal...)
}

func (n NullLogger) LogErrorf(keyVal []any, format string, args ...any) {
	// null implementation
}

func (n NullLogger) LogError(keyVal []any, args ...any) {
	// null implementation
}

func (n NullLogger) LogWarnf(keyVal []any, format string, args ...any) {
	// null implementation
}

func (n NullLogger) LogWarn(keyVal []any, args ...any) {
	// null implementation
}

func (n NullLogger) LogInfof(keyVal []any, format string, args ...any) {
	// null implementation
}

func (n NullLogger) LogInfo(keyVal []any, args ...any) {
	// null implementation
}

func (n NullLogger) LogDebugf(keyVal []any, format string, args ...any) {
	// null implementation
}

func (n NullLogger) LogDebug(keyVal []any, args ...any) {
	// null implementation
}
