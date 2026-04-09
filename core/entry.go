package core

import "time"

// Entry represents a single log event.
// This is the core data structure that flows through the entire system
// created by the Logger, formatted by the Formatter, written by the Sink.
type Entry struct {
	Level     Level          // severity of the log event
	Message   string         // human-readable description
	Timestamp time.Time      // when the event occurred
	Fields    map[string]any // structured key-value pairs
	TraceID   string         // optional distributed trace correlation
}

// NewEntry creates a new Entry with the given level and message.
// Fields is always initialized to avoid nil map panics.
func NewEntry(level Level, msg string) Entry {
	return Entry{
		Level:     level,
		Message:   msg,
		Timestamp: time.Now(),
		Fields:    make(map[string]any),
	}
}
