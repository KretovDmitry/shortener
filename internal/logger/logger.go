// Package logger provides context-aware and structured logging capabilities.
package logger

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/google/uuid"
	sqldblogger "github.com/simukti/sqldb-logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is a logger that supports log levels, context and structured logging.
type Logger interface {
	// With returns a logger based off the root logger
	// and decorates it with the given context and arguments.
	With(ctx context.Context, args ...interface{}) Logger

	// Log implements sqldblogger.Logger interface.
	Log(ctx context.Context, level sqldblogger.Level, msg string, data map[string]interface{})

	// Debug uses fmt.Sprint to construct and log a message at DEBUG level.
	Debug(args ...interface{})
	// Info uses fmt.Sprint to construct and log a message at INFO level.
	Info(args ...interface{})
	// Error uses fmt.Sprint to construct and log a message at ERROR level.
	Error(args ...interface{})

	// Debugf uses fmt.Sprintf to construct and log a message at DEBUG level.
	Debugf(format string, args ...interface{})
	// Infof uses fmt.Sprintf to construct and log a message at INFO level.
	Infof(format string, args ...interface{})
	// Errorf uses fmt.Sprintf to construct and log a message at ERROR level.
	Errorf(format string, args ...interface{})

	// Sync flushes any buffered log entries.
	Sync() error

	// SkipCaller allows skip wrappers in the call stack to log actual
	// caller location.
	SkipCaller(depth int) *Log
}

// Log is a zap sugared logger wrraper with additional functionality.
type Log struct {
	*zap.SugaredLogger
}

// Interface implementation check.
var _ Logger = (*Log)(nil)

type contextKey int

const (
	requestIDKey contextKey = iota
	correlationIDKey
)

// Get creates a new logger using the default configuration.
func New(config *config.Config) *Log {
	sync.OnceFunc(func() {
		stdout := zapcore.AddSync(os.Stdout)

		file := zapcore.AddSync(&lumberjack.Logger{
			Filename:   config.Logger.Path,
			MaxSize:    config.Logger.MaxSizeMB,
			MaxBackups: config.Logger.MaxBackups,
			MaxAge:     config.Logger.MaxAgeDays,
			Compress:   true,
		})

		configLevel, err := zapcore.ParseLevel(config.Logger.Level)
		if err != nil {
			log.Println(
				fmt.Errorf("invalid level, defaulting to INFO: %w", err),
			)
		}

		logLevel := zap.NewAtomicLevelAt(configLevel)

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

	return NewWithZap(zap.L().WithOptions(zap.AddCaller()))
}

// NewWithZap creates a new logger using the preconfigured zap logger.
func NewWithZap(l *zap.Logger) *Log {
	return &Log{l.Sugar()}
}

// SkipCaller allows skip wrappers in the call stack to log actual
// caller location.
func (l *Log) SkipCaller(depth int) *Log {
	return &Log{l.WithOptions(zap.AddCallerSkip(depth))}
}

// NewForTest returns a new logger and the corresponding observed logs
// which can be used in unit tests to verify log entries.
func NewForTest() (*Log, *observer.ObservedLogs) {
	core, recorded := observer.New(zapcore.InfoLevel)
	return NewWithZap(zap.New(core)), recorded
}

// Log implements sqldblogger.Logger.
func (l *Log) Log(_ context.Context, level sqldblogger.Level, msg string, data map[string]interface{}) {
	fields := make([]zap.Field, len(data))
	i := 0

	for k, v := range data {
		if k == "query" {
			if query, ok := v.(string); ok {
				fields[i] = zap.String(k, formatQuery(query))
				i++
				continue
			}
		}
		fields[i] = zap.Any(k, v)
		i++
	}

	switch level {
	case sqldblogger.LevelError:
		l.SkipCaller(1).Desugar().Error(msg, fields...)
	case sqldblogger.LevelInfo:
		l.SkipCaller(1).Desugar().Info(msg, fields...)
	case sqldblogger.LevelDebug:
		l.SkipCaller(1).Desugar().Debug(msg, fields...)
	case sqldblogger.LevelTrace:
		// trace will use zap debug
		l.SkipCaller(1).Desugar().Debug(msg, fields...)
	}
}

// With returns a logger based off the root logger
// and decorates it with the given context and arguments.
//
// If the context contains request ID and/or correlation ID information
// (recorded via WithRequestID() and WithCorrelationID()),
// they will be added to every log message generated by the new logger.
//
// The arguments should be specified as a sequence of name, value pairs with names being strings.
// The arguments will also be added to every log message generated by the logger.
func (l *Log) With(ctx context.Context, args ...interface{}) Logger {
	if ctx != nil {
		if id, ok := ctx.Value(requestIDKey).(string); ok {
			args = append(args, zap.String("request_id", id))
		}
		if id, ok := ctx.Value(correlationIDKey).(string); ok {
			args = append(args, zap.String("correlation_id", id))
		}
	}
	if len(args) > 0 {
		return &Log{l.SugaredLogger.With(args...)}
	}
	return l
}

// WithRequest returns a context which knows
// the request ID and correlation ID in the given request.
func WithRequest(ctx context.Context, req *http.Request) context.Context {
	id := getRequestID(req)
	if id == "" {
		id = uuid.New().String()
	}
	ctx = context.WithValue(ctx, requestIDKey, id)
	if id = getCorrelationID(req); id != "" {
		ctx = context.WithValue(ctx, correlationIDKey, id)
	}
	return ctx
}

// getCorrelationID extracts the correlation ID from the HTTP request.
func getCorrelationID(req *http.Request) string {
	return req.Header.Get("X-Correlation-ID")
}

// getRequestID extracts the correlation ID from the HTTP request.
func getRequestID(req *http.Request) string {
	return req.Header.Get("X-Request-ID")
}

// formatQuery removes tabs and replaces newlines with spaces in the given query string.
func formatQuery(q string) string {
	return strings.ReplaceAll(strings.ReplaceAll(q, "\t", ""), "\n", " ")
}
