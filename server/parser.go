package server

import (
	"bytes"
	"context"
	"math"
	"strconv"
	"time"

	"github.com/Z3-N0/flexlog"
)

// LogEntry represents a single parsed log line.
type LogEntry struct {
	Level     string
	Timestamp time.Time
	Message   string
	TraceID   string
	Fields    map[string]any
	Malformed bool
	Source    string
	Offset    int64
}

// Sets Malformed = true if the line cannot be parsed instead of returning an error, it will still be visible in UI instead of being dropped
func ParseLine(ctx context.Context, logger *flexlog.Logger, line []byte, source string, offset int64) LogEntry {
	entry := LogEntry{
		Source: source,
		Offset: offset,
		Fields: make(map[string]any),
	}

	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		entry.Malformed = true
		return entry
	}

	pos := 1 // skip opening '{'
	n := len(line)
	closed := false

	for pos < n {
		// skip whitespace
		pos = skipSpace(line, pos)
		if pos >= n {
			break
		}
		if line[pos] == '}' {
			closed = true
			break
		}

		// parse key
		key, end, ok := readString(line, pos)
		if !ok {
			entry.Malformed = true
			return entry
		}
		pos = end

		// skip colon
		pos = skipSpace(line, pos)
		if pos >= n || line[pos] != ':' {
			entry.Malformed = true
			return entry
		}
		pos++

		// parse value
		pos = skipSpace(line, pos)
		val, end, ok := readValue(line, pos)
		if !ok {
			entry.Malformed = true
			return entry
		}
		pos = end

		// assign known keys, everything else goes into Fields
		switch key {
		case "level":
			if s, ok := val.(string); ok {
				entry.Level = s
			}
		case "msg":
			if s, ok := val.(string); ok {
				entry.Message = s
			}
		case "trace_id":
			if s, ok := val.(string); ok {
				entry.TraceID = s
			}
		case "ts":
			entry.Timestamp = parseTimestamp(val)
		default:
			entry.Fields[key] = val
		}

		pos = skipSpace(line, pos)
		if pos < n && line[pos] == ',' {
			pos++
		}
	}

	if !closed {
		entry.Malformed = true
		return entry
	}

	return entry
}

// parseTimestamp normalises all supported ts formats to time.Time.
func parseTimestamp(val any) time.Time {
	switch v := val.(type) {
	case int64:
		// distinguish unix sec vs unix milli by magnitude
		if v > 1e12 {
			return time.UnixMilli(v).UTC()
		}
		return time.Unix(v, 0).UTC()
	case float64:
		if v > 1e12 {
			return time.UnixMilli(int64(v)).UTC()
		}
		return time.Unix(int64(v), 0).UTC()
	case string:
		// try RFC3339Nano first, then RFC3339, then Kitchen
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, time.Kitchen} {
			if t, err := time.Parse(layout, v); err == nil {
				if layout == time.Kitchen {
					// Kitchen has no date- anchor to today UTC
					now := time.Now().UTC()
					return time.Date(now.Year(), now.Month(), now.Day(),
						t.Hour(), t.Minute(), 0, 0, time.UTC)
				}
				return t.UTC()
			}
		}
	}
	return time.Time{}
}

func skipSpace(b []byte, pos int) int {
	for pos < len(b) && (b[pos] == ' ' || b[pos] == '\t' || b[pos] == '\n' || b[pos] == '\r') {
		pos++
	}
	return pos
}

// readString reads a JSON-quoted string starting at pos. Returns the unescaped value and the position after the closing quote.
func readString(b []byte, pos int) (string, int, bool) {
	if pos >= len(b) || b[pos] != '"' {
		return "", pos, false
	}
	pos++ // skip opening quote
	var buf []byte
	for pos < len(b) {
		c := b[pos]
		if c == '"' {
			return string(buf), pos + 1, true
		}
		if c == '\\' {
			pos++
			if pos >= len(b) {
				return "", pos, false
			}
			switch b[pos] {
			case '"':
				buf = append(buf, '"')
			case '\\':
				buf = append(buf, '\\')
			case '/':
				buf = append(buf, '/')
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'u':
				// basic \uXXXX, just skips for now
				pos += 4
			}
		} else {
			buf = append(buf, c)
		}
		pos++
	}
	return "", pos, false
}

// readValue reads any JSON value (string, number, bool, null, object, array). Objects and arrays are returned as raw strings since flexlog fields rarely nest deeply.
func readValue(b []byte, pos int) (any, int, bool) {
	if pos >= len(b) {
		return nil, pos, false
	}
	switch b[pos] {
	case '"':
		s, end, ok := readString(b, pos)
		return s, end, ok
	case 't':
		if pos+4 <= len(b) && string(b[pos:pos+4]) == "true" {
			return true, pos + 4, true
		}
	case 'f':
		if pos+5 <= len(b) && string(b[pos:pos+5]) == "false" {
			return false, pos + 5, true
		}
	case 'n':
		if pos+4 <= len(b) && string(b[pos:pos+4]) == "null" {
			return nil, pos + 4, true
		}
	case '{', '[':
		// collect the full nested value as a raw string
		raw, end, ok := readNested(b, pos)
		return raw, end, ok
	default:
		// number
		return readNumber(b, pos)
	}
	return nil, pos, false
}

// readNumber reads an integer or float. Returns int64 when possible, float64 otherwise.
func readNumber(b []byte, pos int) (any, int, bool) {
	start := pos
	isFloat := false
	if pos < len(b) && b[pos] == '-' {
		pos++
	}
	for pos < len(b) && b[pos] >= '0' && b[pos] <= '9' {
		pos++
	}
	if pos < len(b) && (b[pos] == '.' || b[pos] == 'e' || b[pos] == 'E') {
		isFloat = true
		pos++
		for pos < len(b) && (b[pos] >= '0' && b[pos] <= '9' || b[pos] == '+' || b[pos] == '-') {
			pos++
		}
	}
	if pos == start {
		return nil, pos, false
	}
	s := string(b[start:pos])
	if isFloat {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, pos, false
		}
		return f, pos, true
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// too large for int64, fall back to float64
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsInf(f, 0) {
			return nil, pos, false
		}
		return f, pos, true
	}
	return i, pos, true
}

// readNested collects a balanced {…} or […] block as a raw string.
func readNested(b []byte, pos int) (string, int, bool) {
	start := pos
	open := b[pos]
	var close byte
	if open == '{' {
		close = '}'
	} else {
		close = ']'
	}
	depth := 0
	inStr := false
	for pos < len(b) {
		c := b[pos]
		if inStr {
			if c == '\\' {
				pos += 2
				continue
			}
			if c == '"' {
				inStr = false
			}
		} else {
			if c == '"' {
				inStr = true
			} else if c == open {
				depth++
			} else if c == close {
				depth--
				if depth == 0 {
					return string(b[start : pos+1]), pos + 1, true
				}
			}
		}
		pos++
	}
	return "", pos, false
}
