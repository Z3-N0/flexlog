package core

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// We define our own context key type to avoid collisions with other packages.
// If we used a plain string as the key, any package using the same string would overwrite our value. A custom unexported type makes our key unique.
type contextKey string

const traceIDKey contextKey = "trace_id"

// WithTraceID attaches a trace ID to a context so it flows through the call stack automatically.
// Call this once at the start of a request and every log call that receives the context will include the trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext pulls the trace ID back out of a context.
// Returns an empty string if the context has no trace ID attached.
func TraceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(traceIDKey).(string)
	return id
}

// Logger writes structured log entries. Safe to use from multiple goroutines simultaneously — the mutex ensures output lines never interleave.
type Logger struct {
	mu       sync.Mutex     // protects writes to stdout
	minLevel Level          // entries below this level are silently dropped
	fields   map[string]any // fields that appear on every entry from this logger
	exit     func(int)      // defaults to os.Exit, overridable in tests
}

// Option is a function that configures a Logger.
// This is the functional options pattern — it lets New() have sensible defaults while still allowing customization.
type Option func(*Logger)

// WithLevel sets the minimum log level. Entries below this level are silently dropped.
// Defaults to LevelInfo if not specified.
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.minLevel = level
	}
}

// New creates a Logger with sensible defaults — minimum level is LevelInfo.
// Pass options to customize behavior:
//
//	log := New()                        // defaults to LevelInfo
//	log := New(WithLevel(LevelDebug))   // more verbose
//	log := New(WithLevel(LevelDisabled)) // silence everything
func New(opts ...Option) *Logger {
	l := &Logger{
		minLevel: LevelInfo, // sensible default — not too noisy, not too quiet
		fields:   make(map[string]any),
		exit:     os.Exit, // real exit in production
	}

	// apply each option on top of the defaults
	for _, opt := range opts {
		opt(l)
	}

	return l
}

// With returns a new child logger with extra fields permanently attached.
// Every entry from the child will include these fields automatically.
// The original logger is never modified, so this is safe to call concurrently.
// Keys and values must alternate: With("service", "api", "env", "prod")
func (l *Logger) With(keysAndValues ...any) *Logger {
	child := &Logger{
		minLevel: l.minLevel,
		fields:   make(map[string]any, len(l.fields)+len(keysAndValues)/2),
		exit:     l.exit, // child inherits the same exit function
	}

	// carry all fields from the parent into the child
	for k, v := range l.fields {
		child.fields[k] = v
	}

	// add the new fields on top
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		child.fields[key] = keysAndValues[i+1]
	}

	return child
}

// log is the single path all entries flow through. It checks the level, builds the entry, merges fields, and writes output.
// The formatted output here is temporary — the JSON formatter replaces this.
func (l *Logger) log(ctx context.Context, level Level, msg string, keysAndValues ...any) {
	if level < l.minLevel {
		return
	}

	entry := NewEntry(level, msg)
	entry.TraceID = TraceIDFromContext(ctx)

	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		entry.Fields[key] = keysAndValues[i+1]
	}

	l.mu.Lock()
	fmt.Fprintf(os.Stdout, "[%s] %s %v\n", entry.Level, entry.Message, entry.Fields)
	l.mu.Unlock()
}

// Trace logs fine-grained details, usually only useful when debugging a specific problem. Disabled in most production environments.
func (l *Logger) Trace(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelTrace, msg, keysAndValues...)
}

// Debug logs information useful during development and troubleshooting.
func (l *Logger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelDebug, msg, keysAndValues...)
}

// Info logs normal operational events — server started, request received, etc.
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

// Fatal logs the entry and immediately terminates the program.
// Deferred functions will not run. Use only for unrecoverable startup failures.
func (l *Logger) Fatal(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelFatal, msg, keysAndValues...)
	l.exit(1)
}
