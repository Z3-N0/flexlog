package core

import "time"

type TimeFormat int8

const (
	TimeUnixMilli   TimeFormat = iota // 1775315926040 (default)
	TimeUnixSec                       // 1775315926
	TimeRFC3339                       // "2026-04-04T10:30:00Z"
	TimeRFC3339Nano                   // "2026-04-04T10:30:00.000000000Z"
	TimeKitchen                       // "3:04PM"
)

// FormatTime returns the timestamp in the configured format.
// Returns either a string or int64 depending on the format
func FormatTime(t time.Time, fmt TimeFormat) any {
	switch fmt {
	case TimeUnixSec:
		return t.Unix()
	case TimeRFC3339:
		return t.UTC().Format(time.RFC3339)
	case TimeRFC3339Nano:
		return t.UTC().Format(time.RFC3339Nano)
	case TimeKitchen:
		return t.Format(time.Kitchen)
	default: // TimeUnixMilli
		return t.UnixMilli()
	}
}
