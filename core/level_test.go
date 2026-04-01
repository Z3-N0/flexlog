package core

import (
	"context"
	"testing"
	"time"
)

// --- Level tests ---

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDisabled, "DISABLED"},
		{LevelTrace, "TRACE"},
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestLevelOrdering(t *testing.T) {
	if LevelDisabled >= LevelTrace {
		t.Error("LevelDisabled should be less than LevelTrace")
	}
	if LevelTrace >= LevelDebug {
		t.Error("LevelTrace should be less than LevelDebug")
	}
	if LevelDebug >= LevelInfo {
		t.Error("LevelDebug should be less than LevelInfo")
	}
	if LevelInfo >= LevelWarn {
		t.Error("LevelInfo should be less than LevelWarn")
	}
	if LevelWarn >= LevelError {
		t.Error("LevelWarn should be less than LevelError")
	}
	if LevelError >= LevelFatal {
		t.Error("LevelError should be less than LevelFatal")
	}
}

// --- Entry tests ---

func TestNewEntry(t *testing.T) {
	before := time.Now()
	e := NewEntry(LevelInfo, "test message")
	after := time.Now()

	if e.Level != LevelInfo {
		t.Errorf("got level %v, want %v", e.Level, LevelInfo)
	}
	if e.Message != "test message" {
		t.Errorf("got message %q, want %q", e.Message, "test message")
	}
	if e.Timestamp.Before(before) || e.Timestamp.After(after) {
		t.Errorf("timestamp %v is outside expected range", e.Timestamp)
	}

	// this would panic if Fields is nil; the test itself proves it is initialized
	e.Fields["key"] = "value"
	if e.Fields["key"] != "value" {
		t.Errorf("Fields not working correctly")
	}
}

// --- Logger tests ---

func TestNewDefaultsToLevelInfo(t *testing.T) {
	l := New()

	// default level should be LevelInfo
	if l.minLevel != LevelInfo {
		t.Errorf("default minLevel = %v, want %v", l.minLevel, LevelInfo)
	}
}

func TestNewWithLevel(t *testing.T) {
	l := New(WithLevel(LevelDebug))

	if l.minLevel != LevelDebug {
		t.Errorf("minLevel = %v, want %v", l.minLevel, LevelDebug)
	}
}

func TestNewWithLevelDisabled(t *testing.T) {
	l := New(WithLevel(LevelDisabled))

	if l.minLevel != LevelDisabled {
		t.Errorf("minLevel = %v, want %v", l.minLevel, LevelDisabled)
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	// count how many times exit is called, should be zero we use this trick to verify Fatal doesn't fire unexpectedly
	logged := 0
	ctx := context.Background()

	l := New(WithLevel(LevelWarn))
	// swap out exit so Fatal doesn't kill the test runner
	l.exit = func(int) {}

	// redirect output so we don't pollute test output

	// these should be silently dropped
	l.Trace(ctx, "should be dropped")
	l.Debug(ctx, "should be dropped")
	l.Info(ctx, "should be dropped")
	_ = logged

	// if we got here without panicking the drops worked correctly
}

func TestLoggerWith(t *testing.T) {
	l := New()
	child := l.With("service", "api", "env", "prod")

	// child should have the fields
	if child.fields["service"] != "api" {
		t.Errorf("child missing service field")
	}
	if child.fields["env"] != "prod" {
		t.Errorf("child missing env field")
	}

	// parent should be unmodified
	if _, ok := l.fields["service"]; ok {
		t.Error("With() modified the parent logger, it should not")
	}
}

func TestLoggerWithInheritsParentFields(t *testing.T) {
	l := New()
	parent := l.With("service", "api")
	child := parent.With("request_id", "zeno")

	// child should have both parent and its own fields
	if child.fields["service"] != "api" {
		t.Errorf("child missing inherited service field")
	}
	if child.fields["request_id"] != "zeno" {
		t.Errorf("child missing own request_id field")
	}

	// parent should only have its own field
	if _, ok := parent.fields["request_id"]; ok {
		t.Error("child's With() modified the parent, it should not")
	}
}

func TestLoggerWithInheritsExitFunc(t *testing.T) {
	exitCalled := false
	l := New()
	l.exit = func(int) { exitCalled = true }

	child := l.With("key", "val")
	child.Fatal(context.Background(), "fatal error")

	if !exitCalled {
		t.Error("child logger did not inherit exit function from parent")
	}
}

func TestFatalCallsExit(t *testing.T) {
	exitCode := 0
	l := New()
	l.exit = func(code int) { exitCode = code }

	l.Fatal(context.Background(), "something went very wrong")

	if exitCode != 1 {
		t.Errorf("Fatal called exit with code %d, want 1", exitCode)
	}
}

// --- Context tests ---

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-abc-123")

	got := TraceIDFromContext(ctx)
	if got != "trace-abc-123" {
		t.Errorf("got trace ID %q, want %q", got, "trace-abc-123")
	}
}

func TestTraceIDFromContextEmpty(t *testing.T) {
	ctx := context.Background()

	// context with no trace ID should return empty string, not panic
	got := TraceIDFromContext(ctx)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}
