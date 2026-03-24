package core

import (
	"testing"
	"time"
)

func TestLevelString(t *testing.T){
	tests := []struct{
		level Level
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

	e.Fields["key"] = "value"
	if e.Fields["key"] != "value" {
		t.Errorf("Fields not working correctly")
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
