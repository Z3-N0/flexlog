package core

import "time"

type Level int8

const(
	LevelDisabled Level = iota - 1
	LevelTrace
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)


func(l Level) String() string{
	switch l{
	case LevelDisabled:
		return "DISABLED"
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Entry struct{
	Level Level
	Message string
	Timestamp time.Time
	Fields map[string]any
	ServiceID string
	TraceID string
}


func NewEntry(level Level, msg string) Entry {
	return Entry{
		Level:     level,
		Message:   msg,
		Timestamp: time.Now(),
		Fields:    make(map[string]any),
	}
}
