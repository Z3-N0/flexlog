package core

import (
	"context"
	"fmt"
	"maps"
	"os"
	"sync"

	"github.com/Z3-N0/flexlog/sinks"
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
	mu        sync.Mutex     // protects writes to all sinks
	minLevel  Level          // entries below this level are silently dropped
	fields    map[string]any // fields that appear on every entry from this logger
	exit      func(int)      // defaults to os.Exit, overridable in tests
	sinks     []sinks.Sink   // destinations this logger writes to
	timeFmt   TimeFormat     // controls how timestamps are serialized
	fatalHook FatalHook      // controls how fatal call behaves
}

// This is the functional options pattern it lets New() have defaults while still allowing customization.
type Option func(*Logger)

// WithLevel sets the minimum log level. Entries below this level are silently dropped (Default is LevelInfo)
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.minLevel = level
	}
}

// WithSink adds a destination for log entries. Call multiple times to write to multiple destinations.
func WithSink(s sinks.Sink) Option {
	return func(l *Logger) {
		l.sinks = append(l.sinks, s)
	}
}

// WithTimeFormat sets the timestamp format for all entries. Defaults to TimeUnixMilli.
func WithTimeFormat(fmt TimeFormat) Option {
	return func(l *Logger) {
		l.timeFmt = fmt
	}
}

// WithFatalHook controls what happens after a Fatal log entry is written. Defaults to FatalHookExit.
func WithFatalHook(hook FatalHook) Option {
	return func(l *Logger) {
		l.fatalHook = hook
	}
}

// New creates a Logger with defaults, minimum level is LevelInfo.

// log := New()                                         // defaults to LevelInfo, stdout
// log := New(WithLevel(LevelDebug))                    // minLevel is now LevelDebug
// log := New(WithSink(sinks.Stdout))                   // explicit stdout
// log := New(WithSink(sinks.Stdout), WithSink(file))   // stdout and file
func New(opts ...Option) *Logger {
	l := &Logger{
		minLevel: LevelInfo,
		fields:   make(map[string]any),
		exit:     os.Exit, // real exit in production
		timeFmt:  TimeUnixMilli,
	}
	// Apply each option on top of the defaults
	for _, opt := range opts {
		opt(l)
	}
	// If no sinks provided, default to stdout so existing behavior is unchanged
	if len(l.sinks) == 0 {
		l.sinks = append(l.sinks, sinks.Stdout)
	}
	return l
}

// With returns a new child logger with extra fields permanently attached.
// Every entry from the child will include these fields.
// Keys and values must alternate: With("service", "api", "env", "prod").
func (l *Logger) With(keysAndValues ...any) *Logger {
	if len(keysAndValues) == 0 {
		return l
	}
	child := &Logger{
		minLevel:  l.minLevel,
		fields:    make(map[string]any, len(l.fields)+len(keysAndValues)/2),
		exit:      l.exit, // child inherits the same exit function
		sinks:     l.sinks,
		timeFmt:   l.timeFmt,
		fatalHook: l.fatalHook,
	}
	// Carry all fields from the parent into the child.
	maps.Copy(child.fields, l.fields)
	// Add the new fields on top.
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		child.fields[key] = keysAndValues[i+1]
	}
	// to handle missing values in key-value pair
	if len(keysAndValues)%2 != 0 {
		key := fmt.Sprintf("%v", keysAndValues[len(keysAndValues)-1])
		child.fields[key] = "MISSING"
	}
	return child
}

// The single path all entries flow through. It checks the level, builds the entry, merges fields, and writes to all sinks.
func (l *Logger) log(ctx context.Context, level Level, msg string, keysAndValues ...any) {
	if level < l.minLevel {
		return
	}
	entry := NewEntry(level, msg)
	if id := TraceIDFromContext(ctx); id != "" {
		entry.TraceID = id
	}
	maps.Copy(entry.Fields, l.fields)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		entry.Fields[key] = keysAndValues[i+1]
	}
	if len(keysAndValues)%2 != 0 {
		key := fmt.Sprintf("%v", keysAndValues[len(keysAndValues)-1])
		entry.Fields[key] = "MISSING"
	}

	ts := FormatTime(entry.Timestamp, l.timeFmt)

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, sink := range l.sinks {
		if err := sink.Write(entry.Level.String(), ts, entry.TraceID, entry.Message, entry.Fields); err != nil {
			os.Stderr.Write([]byte("flexlog: sink write error: " + err.Error() + "\n"))
		}
	}
}

// Close flushes and closes all sinks. Always defer this after creating a logger.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var firstErr error
	for _, sink := range l.sinks {
		if err := sink.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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

// Fatal logs the entry and exit behaviour is dependent on hook used. Defaults to exit. Use only for unrecoverable failures.
func (l *Logger) Fatal(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(ctx, LevelFatal, msg, keysAndValues...)
	switch l.fatalHook {
	case FatalHookPanic:
		panic("flexlog: fatal")
	case FatalHookNoop:
		// do nothing
	default:
		l.exit(1)
	}
}
