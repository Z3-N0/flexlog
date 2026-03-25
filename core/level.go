package core

// Level represents the severity of a log entry.
type Level int8

const (
	LevelDisabled Level = iota - 1 // LevelDisabled silences all log output
	LevelTrace                     // LevelTrace is the most verbose level
	LevelDebug                     // LevelDebug is for development details
	LevelInfo                      // LevelInfo is for general operational messages
	LevelWarn                      // LevelWarn is for non-critical problems
	LevelError                     // LevelError is for failures that need attention
	LevelFatal                     // LevelFatal logs and then terminates the program
)

// String returns the human-readable name of the level.
// This is what appears in the "level" field of the JSON output.
func (l Level) String() string {
	switch l {
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
