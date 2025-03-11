// Package logger provides a structured logging system for the Document Management Platform.
// It implements a flexible, level-based logging mechanism with JSON output format,
// context awareness, and integration with distributed tracing.
// This package ensures consistent logging across all components of the application.
package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap" // v1.24.0+
	"go.uber.org/zap/zapcore" // v1.24.0+
)

// Global variables
var (
	logger          *zap.Logger
	initialized     bool
	defaultLogLevel = zapcore.InfoLevel
)

// Context keys for request metadata
const (
	contextKeyRequestID = "request_id"
	contextKeyTraceID   = "trace_id"
	contextKeySpanID    = "span_id"
)

// LogConfig defines the configuration options for the logger
type LogConfig struct {
	// Level sets the minimum log level: "debug", "info", "warn", "error"
	Level string
	// Format sets the output format: "json" or "console"
	Format string
	// Output determines where logs are written: "console" or "file"
	Output string
	// FilePath specifies the log file path when Output is "file"
	FilePath string
	// Development enables development mode with more verbose output
	Development bool
}

// Init initializes the logger with the specified configuration
func Init(config LogConfig) error {
	// Check if already initialized
	if initialized {
		return nil
	}

	// Create encoder configuration with JSON format and RFC3339 timestamp format
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set log level based on configuration
	var level zapcore.Level
	switch config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = defaultLogLevel
	}

	// Create core with encoder and appropriate output (console or file)
	var output zapcore.WriteSyncer
	if config.Output == "file" && config.FilePath != "" {
		file, err := os.OpenFile(config.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		output = zapcore.AddSync(file)
	} else {
		output = zapcore.AddSync(os.Stdout)
	}

	// Create encoder based on format
	var encoder zapcore.Encoder
	if config.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create zap logger with core
	core := zapcore.NewCore(encoder, output, zap.NewAtomicLevelAt(level))
	zapLogger := zap.New(core, zap.AddCaller())
	
	if config.Development {
		zapLogger = zapLogger.WithOptions(zap.Development())
	}

	// Set global logger instance
	logger = zapLogger
	initialized = true

	// Log successful initialization
	logger.Info("Logger initialized successfully", 
		zap.String("level", config.Level), 
		zap.String("format", config.Format),
		zap.String("output", config.Output))

	return nil
}

// Shutdown flushes any buffered log entries and releases resources
func Shutdown() error {
	// Check if initialized, if not return nil
	if !initialized {
		return nil
	}
	
	// Sync the logger to flush any buffered logs
	err := logger.Sync()
	
	// Set initialized to false
	initialized = false
	
	// Return any error from sync
	return err
}

// Debug logs a message at debug level
func Debug(msg string, fields ...interface{}) {
	// Check if initialized, if not return
	if !initialized {
		return
	}
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at debug level with fields
	logger.Debug(msg, zapFields...)
}

// Info logs a message at info level
func Info(msg string, fields ...interface{}) {
	// Check if initialized, if not return
	if !initialized {
		return
	}
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at info level with fields
	logger.Info(msg, zapFields...)
}

// Warn logs a message at warn level
func Warn(msg string, fields ...interface{}) {
	// Check if initialized, if not return
	if !initialized {
		return
	}
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at warn level with fields
	logger.Warn(msg, zapFields...)
}

// Error logs a message at error level
func Error(msg string, fields ...interface{}) {
	// Check if initialized, if not return
	if !initialized {
		return
	}
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at error level with fields
	logger.Error(msg, zapFields...)
}

// WithContext creates a logger with context values (request ID, trace ID, span ID)
func WithContext(ctx context.Context) *zap.Logger {
	// Check if initialized, if not return no-op logger
	if !initialized {
		return zap.NewNop()
	}
	
	// Check if context is nil, if so return original logger
	if ctx == nil {
		return logger
	}
	
	// Extract request ID from context if present
	var fields []zap.Field
	if requestID, ok := ctx.Value(contextKeyRequestID).(string); ok && requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	
	// Extract trace ID from context if present
	if traceID, ok := ctx.Value(contextKeyTraceID).(string); ok && traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	
	// Extract span ID from context if present
	if spanID, ok := ctx.Value(contextKeySpanID).(string); ok && spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}
	
	// Return logger with added fields
	return logger.With(fields...)
}

// WithField creates a logger with an additional field
func WithField(key string, value interface{}) *zap.Logger {
	// Check if initialized, if not return no-op logger
	if !initialized {
		return zap.NewNop()
	}
	
	// Convert key-value pair to zap field
	field := zap.Any(key, value)
	
	// Return logger with added field
	return logger.With(field)
}

// WithFields creates a logger with additional fields
func WithFields(fields map[string]interface{}) *zap.Logger {
	// Check if initialized, if not return no-op logger
	if !initialized {
		return zap.NewNop()
	}
	
	// Convert map to zap fields
	zapFields := mapToZapFields(fields)
	
	// Return logger with added fields
	return logger.With(zapFields...)
}

// WithError creates a logger with error details
func WithError(err error) *zap.Logger {
	// Check if initialized, if not return no-op logger
	if !initialized {
		return zap.NewNop()
	}
	
	// Check if error is nil, if so return original logger
	if err == nil {
		return logger
	}
	
	// Add error message as field
	fields := []zap.Field{zap.String("error", err.Error())}
	
	// If error has stack trace, add it as field
	type stackTracer interface {
		StackTrace() []byte
	}
	if st, ok := err.(stackTracer); ok {
		fields = append(fields, zap.ByteString("stacktrace", st.StackTrace()))
	}
	
	// Return logger with error fields
	return logger.With(fields...)
}

// DebugContext logs a debug message with context values
func DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	// Create logger with context
	ctxLogger := WithContext(ctx)
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at debug level with fields
	ctxLogger.Debug(msg, zapFields...)
}

// InfoContext logs an info message with context values
func InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	// Create logger with context
	ctxLogger := WithContext(ctx)
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at info level with fields
	ctxLogger.Info(msg, zapFields...)
}

// WarnContext logs a warn message with context values
func WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	// Create logger with context
	ctxLogger := WithContext(ctx)
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at warn level with fields
	ctxLogger.Warn(msg, zapFields...)
}

// ErrorContext logs an error message with context values
func ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	// Create logger with context
	ctxLogger := WithContext(ctx)
	
	// Convert fields to zap fields
	zapFields := fieldsToZapFields(fields...)
	
	// Log message at error level with fields
	ctxLogger.Error(msg, zapFields...)
}

// fieldsToZapFields converts a list of fields to zap fields
func fieldsToZapFields(fields ...interface{}) []zap.Field {
	// Initialize empty zap fields array
	zapFields := make([]zap.Field, 0, len(fields)/2)
	
	// Process fields in pairs (key, value)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if !ok {
				continue // Skip if key is not a string
			}
			zapFields = append(zapFields, zap.Any(key, fields[i+1]))
		}
	}
	
	// Return array of zap fields
	return zapFields
}

// mapToZapFields converts a map to zap fields
func mapToZapFields(fields map[string]interface{}) []zap.Field {
	// Initialize empty zap fields array
	zapFields := make([]zap.Field, 0, len(fields))
	
	// Iterate through map entries
	for k, v := range fields {
		// Convert each entry to appropriate zap field type
		zapFields = append(zapFields, zap.Any(k, v))
	}
	
	// Return array of zap fields
	return zapFields
}