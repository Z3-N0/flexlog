package server

import (
	"fmt"
	"strconv"
	"time"
)

// parseTimeParam parses a time value from a URL query parameter.
func ParseTimeParam(s string) (time.Time, error) {
	// 1. Handle Unix Timestamps (Keep this as is)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		if n > 1e12 {
			return time.UnixMilli(n).UTC(), nil
		}
		return time.Unix(n, 0).UTC(), nil
	}

	// 2. Define the layouts to try
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05", // HTML5 datetime-local (with seconds)
		"2006-01-02T15:04",    // HTML5 datetime-local (without seconds)
	}

	for _, layout := range layouts {
		// Use time.Parse if the string has a timezone or time.ParseInLocation if it doesn't.
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognised time format: %q", s)
}
