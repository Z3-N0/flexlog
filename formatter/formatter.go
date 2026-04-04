package formatter

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// bufPool reuses byte buffers across calls to avoid repeated heap allocations.
var bufPool = sync.Pool{
	New: func() any { return &bytes.Buffer{} },
}

// Format serializes a log entry to JSON bytes.
// Uses a pooled buffer and manual writing to avoid reflection and minimize allocations.
func Format(level string, ts time.Time, traceID string, msg string, fields map[string]any) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	buf.WriteByte('{')

	// fixed fields first, in a consistent order
	writeStringField(buf, "level", level, true)
	buf.WriteByte(',')
	writeInt(buf, "ts", ts.UnixMilli())
	if traceID != "" {
		writeStringField(buf, "trace_id", traceID, false)
	}
	writeStringField(buf, "msg", msg, false)

	// dynamic fields last
	for k, v := range fields {
		buf.WriteByte(',')
		switch val := v.(type) {
		case string:
			writeString(buf, k, val)
		case int:
			writeInt(buf, k, int64(val))
		case int64:
			writeInt(buf, k, val)
		case float64:
			writeFloat(buf, k, val)
		case bool:
			writeBool(buf, k, val)
		default:
			writeString(buf, k, stringify(v))
		}
	}

	buf.WriteByte('}')

	// Final allocation to return the result
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())

	return result, nil
}

// --- Internal Helpers ---

// Separate Because it has an additional Check
func writeStringField(buf *bytes.Buffer, key, val string, first bool) {
	if !first {
		buf.WriteByte(',')
	}
	writeString(buf, key, val)
}

func writeString(buf *bytes.Buffer, key, val string) {
	buf.WriteByte('"')
	writeEscaped(buf, key)
	buf.WriteString(`":"`)
	writeEscaped(buf, val)
	buf.WriteByte('"')
}

func writeInt(buf *bytes.Buffer, key string, val int64) {
	buf.WriteByte('"')
	writeEscaped(buf, key)
	buf.WriteString(`":`)
	// Use a stack-allocated byte slice to avoid heap allocation
	var b [20]byte
	buf.Write(strconv.AppendInt(b[:0], val, 10))
}

func writeFloat(buf *bytes.Buffer, key string, val float64) {
	buf.WriteByte('"')
	writeEscaped(buf, key)
	buf.WriteString(`":`)
	var b [32]byte
	buf.Write(strconv.AppendFloat(b[:0], val, 'f', -1, 64))
}

func writeBool(buf *bytes.Buffer, key string, val bool) {
	buf.WriteByte('"')
	writeEscaped(buf, key)
	buf.WriteString(`":`)
	if val {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
}

// writeEscaped writes the string to the buffer while escaping JSON special characters.
// This avoids the extra allocation of creating a new "escaped" string.
func writeEscaped(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			buf.WriteByte(s[i])
		}
	}
}

func stringify(v any) string {
	return fmt.Sprintf("%v", v)
}
