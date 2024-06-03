// Package logger provides a logger using the zap library.
package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"

	"github.com/KretovDmitry/shortener/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// loggerCtxKey is a type used to store the logger in the context.
type loggerCtxKey struct{}

// Get returns the default zap logger.
// It initializes the logger with the specified log level from the config package.
// If the log level is invalid, it defaults to INFO.
// The logger logs to both the console and a file named "logs/app.log".
func Get() *zap.Logger {
	sync.OnceFunc(func() {
		stdout := zapcore.AddSync(os.Stdout)

		file := zapcore.AddSync(&lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    5,  // megabytes
			MaxBackups: 10, // max num of old log files
			MaxAge:     14, // days
			Compress:   true,
		})

		levelFromEnv, err := zapcore.ParseLevel(config.LogLevel)
		if err != nil {
			log.Println(
				fmt.Errorf("invalid level, defaulting to INFO: %w", err),
			)
		}

		logLevel := zap.NewAtomicLevelAt(levelFromEnv)

		productionCfg := zap.NewProductionEncoderConfig()
		productionCfg.TimeKey = "timestamp"
		productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder

		developmentCfg := zap.NewDevelopmentEncoderConfig()
		developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

		consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)
		fileEncoder := zapcore.NewJSONEncoder(productionCfg)

		var gitRevision string

		buildInfo, ok := debug.ReadBuildInfo()
		if ok {
			for _, v := range buildInfo.Settings {
				if v.Key == "vcs.revision" {
					gitRevision = v.Value
					break
				}
			}
		}

		// log to multiple destinations (console and file)
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, stdout, logLevel),
			zapcore.NewCore(fileEncoder, file, logLevel).
				With(
					[]zapcore.Field{
						zap.String("git_revision", gitRevision),
						zap.String("go_version", buildInfo.GoVersion),
					},
				),
		)

		zap.ReplaceGlobals(zap.New(core))
	})()

	return zap.L()
}

// FromCtx returns the Logger associated with the ctx.
// If no logger is associated, the default logger is returned.
func FromCtx(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerCtxKey{}).(*zap.Logger); ok {
		return l
	}

	return Get()
}

// WithCtx returns a copy of ctx with the Logger attached.
func WithCtx(ctx context.Context, l *zap.Logger) context.Context {
	if lp, ok := ctx.Value(loggerCtxKey{}).(*zap.Logger); ok {
		if lp == l {
			// Do not store the same logger.
			return ctx
		}
	}

	return context.WithValue(ctx, loggerCtxKey{}, l)
}
