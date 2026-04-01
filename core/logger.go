package core

import (
	"context"
	"fmt"
	"maps"
	"os"
	"sync"
)

// Context key is defined as its own type to avoid collision with other packages.
type contextKey string

const traceIDKey contextKey = "trace_id"

// WithTraceID attaches a trace ID to a context so it flows through the call stack automatically.
// Call this once at the start of a request and every log call that receives the context will include the trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// Pulls the trace ID back out of a context or returns an empty string if no trace ID is attached in context.
func TraceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(traceIDKey).(string)
	return id
}

// This is the structure of all log entries. Safe to use from multiple goroutines simultaneously, the mutex ensures output lines never interleave.
type Logger struct {
	mu       sync.Mutex     // protects writes to stdout
	minLevel Level          // entries below this level are silently dropped
	fields   map[string]any // fields that appear on every entry from this logger
	exit     func(int)      // defaults to os.Exit, overridable in tests
}

// function that configures a Logger.
// This is the functional options pattern it lets New() have defaults while still allowing customization.
type Option func(*Logger)

// WithLevel sets the minimum log level. Entries below this level are silently dropped (Default is LevelInfo)
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.minLevel = level
	}
}

// New creates a Logger with defaults, minimum level is LevelInfo.
// Pass options to customize behavior:
//
//	log := New()                        // defaults to LevelInfo
//	log := New(WithLevel(LevelDebug))   // minLevel is now LevelDebug
func New(opts ...Option) *Logger {
	l := &Logger{
		minLevel: LevelInfo,
		fields:   make(map[string]any),
		exit:     os.Exit, // real exit in production
	}

	// Apply each option on top of the defaults
	for _, opt := range opts {
		opt(l)
	}

	return l
}

// With returns a new child logger with extra fields permanently attached.
// Every entry from the child will include these fields.
// Keys and values must alternate: With("service", "api", "env", "prod").
func (l *Logger) With(keysAndValues ...any) *Logger {
	child := &Logger{
		minLevel: l.minLevel,
		fields:   make(map[string]any, len(l.fields)+len(keysAndValues)/2),
		exit:     l.exit, // child inherits the same exit function
	}

	// Carry all fields from the parent into the child.
	maps.Copy(child.fields, l.fields)

	// Add the new fields on top.
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		child.fields[key] = keysAndValues[i+1]
	}

	return child
}

// The single path all entries flow through. It checks the level, builds the entry, merges fields, and writes output.
// The formatted output here is temporary, the JSON formatter will replace this.
func (l *Logger) log(ctx context.Context, level Level, msg string, keysAndValues ...any) {
	if level < l.minLevel {
		return
	}

	entry := NewEntry(level, msg)
	entry.TraceID = TraceIDFromContext(ctx)

	maps.Copy(entry.Fields, l.fields)

	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		entry.Fields[key] = keysAndValues[i+1]
	}

	l.mu.Lock()
	fmt.Fprintf(os.Stdout, "[%s] %s %v\n", entry.Level, entry.Message, entry.Fields)
	l.mu.Unlock()
}

// Trace logs fine-grained details, usually only useful when debugging a specific problem. Disabled in most environments.
func (l *Logger) Trace(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelTrace, msg, keysAndValues...)
}

// Debug logs information useful during development and troubleshooting.
func (l *Logger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelDebug, msg, keysAndValues...)
}

// Info logs normal operational events - server started, request received, etc.
func (l *Logger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelInfo, msg, keysAndValues...)
}

// Warn logs something unexpected that the system recovered from, but that someone should probably look at.
func (l *Logger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelWarn, msg, keysAndValues...)
}

// Error logs a failure that needs attention but didn't crash the program.
func (l *Logger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelError, msg, keysAndValues...)
}

// Fatal logs the entry and immediately terminates the program. Use only for unrecoverable failures.
func (l *Logger) Fatal(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelFatal, msg, keysAndValues...)
	l.exit(1)
}
