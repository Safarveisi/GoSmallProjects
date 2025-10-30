package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a thin wrapper that holds both the raw zap.Logger and its
// "Sugared" counterpart for convenience.
type Logger struct {
	*zap.Logger
	*zap.SugaredLogger
}

// New creates a new logger based on the provided log level string.
// Accepted levels (case-insensitive): "debug", "info", "warn", "error".
//
// The returned *Logger contains both the classic *zap.Logger and a
// SugaredLogger (which allows the familiar `Infof`, `Errorf` … style).
func New(level string) (*Logger, error) {
	// Parse level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		// Return the error so the caller can decide to abort or fall‑back.
		return nil, err
	}

	// Encoder configuration - JSON, ISO-8601 timestamps, capital level
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	// Core - write JSON to stdout (or stderr if you prefer)
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.Lock(zapcore.AddSync(os.Stdout)), // no nil logger
		zapLevel,
	)

	// Build the logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := zapLogger.Sugar()

	return &Logger{
		Logger:        zapLogger,
		SugaredLogger: sugar,
	}, nil
}

// FromContext extracts a *zap.Logger that may have been stored in the context.
// If none is present, the fallback logger is returned.
func FromContext(ctx context.Context, fallback *Logger) *zap.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return fallback.Logger
}

// WithContext returns a new context that carries the supplied logger.
// This is handy for HTTP middlewares where you want request-scoped fields
// (e.g., request ID, user, etc.).
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// loggerKey is an unexported type to avoid key collisions in context.
type loggerKey struct{}

// WithRequestID returns a copy of the logger with a request-id field attached.
// Typical usage in an HTTP middleware:
//
//	func requestIDMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        id := uuid.NewString()
//	        l := logger.FromContext(r.Context(), defaultLogger).With(zap.String("req_id", id))
//	        ctx := logger.WithContext(r.Context(), l)
//	        next.ServeHTTP(w, r.WithContext(ctx))
//	    })
//	}
func WithRequestID(l *zap.Logger, reqID string) *zap.Logger {
	return l.With(zap.String("req_id", reqID))
}

// Flush forces any buffered log entries to be written.
// Call this from `main` just before the program exits.
func Flush(l *zap.Logger) {
	// Sync returns any error encountered while flushing. In many cases we
	// can safely ignore it, but logging the error helps during debugging.
	if err := l.Sync(); err != nil {
		// zap's Sync can return `sync: invalid argument` on Windows when the
		// logger has no file output. That is harmless, so we ignore it.
	}
}
