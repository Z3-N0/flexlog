package server

import (
	"context"
	"testing"
	"time"

	"github.com/Z3-N0/flexlog"
)

// newLogger returns a silent logger suitable for tests.
func newLogger() *flexlog.Logger {
	return flexlog.New(flexlog.WithLevel(flexlog.LevelTrace))
}

// ── ParseLine ────────────────────────────────────────────────────────────────

func TestParseLine_WellFormed(t *testing.T) {
	ctx := context.Background()
	logger := newLogger()
	defer logger.Close()

	t.Run("all known fields present", func(t *testing.T) {
		line := `{"level":"INFO","ts":1776784434001,"msg":"indexing complete","trace_id":"abc-123"}`
		entry := ParseLine(ctx, logger, []byte(line), "test.log", 42)

		if entry.Malformed {
			t.Fatal("expected well-formed entry")
		}
		if entry.Level != "INFO" {
			t.Errorf("Level = %q, want INFO", entry.Level)
		}
		if entry.Message != "indexing complete" {
			t.Errorf("Message = %q, want \"indexing complete\"", entry.Message)
		}
		if entry.TraceID != "abc-123" {
			t.Errorf("TraceID = %q, want abc-123", entry.TraceID)
		}
		if entry.Source != "test.log" {
			t.Errorf("Source = %q, want test.log", entry.Source)
		}
		if entry.Offset != 42 {
			t.Errorf("Offset = %d, want 42", entry.Offset)
		}
		// unix-milli 1776784434001 → a non-zero timestamp
		if entry.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero for unix-milli ts")
		}
		wantMs := time.UnixMilli(1776784434001).UTC()
		if !entry.Timestamp.Equal(wantMs) {
			t.Errorf("Timestamp = %v, want %v", entry.Timestamp, wantMs)
		}
	})

	t.Run("extra whitespace around tokens", func(t *testing.T) {
		line := `  {  "level" : "DEBUG" , "msg" : "hello world" }  `
		entry := ParseLine(ctx, logger, []byte(line), "test.log", 0)

		if entry.Malformed {
			t.Fatal("expected well-formed entry")
		}
		if entry.Level != "DEBUG" {
			t.Errorf("Level = %q, want DEBUG", entry.Level)
		}
		if entry.Message != "hello world" {
			t.Errorf("Message = %q, want \"hello world\"", entry.Message)
		}
	})

	t.Run("empty object", func(t *testing.T) {
		entry := ParseLine(ctx, logger, []byte(`{}`), "test.log", 0)
		if entry.Malformed {
			t.Fatal("empty object should be well-formed")
		}
	})

	t.Run("unknown keys land in Fields", func(t *testing.T) {
		line := `{"level":"WARN","msg":"ok","latency_ms":123,"retried":true,"ratio":1.5,"meta":{"k":"v"},"tags":["a","b"]}`
		entry := ParseLine(ctx, logger, []byte(line), "test.log", 0)

		if entry.Malformed {
			t.Fatal("expected well-formed entry")
		}
		if v, ok := entry.Fields["latency_ms"]; !ok || v != int64(123) {
			t.Errorf("Fields[latency_ms] = %v (%T), want int64(123)", v, v)
		}
		if v, ok := entry.Fields["retried"]; !ok || v != true {
			t.Errorf("Fields[retried] = %v, want true", v)
		}
		if v, ok := entry.Fields["ratio"]; !ok || v != float64(1.5) {
			t.Errorf("Fields[ratio] = %v, want 1.5", v)
		}
		if _, ok := entry.Fields["meta"]; !ok {
			t.Error("Fields[meta] should be present (nested object)")
		}
		if _, ok := entry.Fields["tags"]; !ok {
			t.Error("Fields[tags] should be present (array)")
		}
	})

	t.Run("null value in field does not panic", func(t *testing.T) {
		line := `{"level":"INFO","msg":"ok","ctx":null}`
		entry := ParseLine(ctx, logger, []byte(line), "test.log", 0)
		if entry.Malformed {
			t.Fatal("null field value should not mark entry as malformed")
		}
		if v, ok := entry.Fields["ctx"]; !ok || v != nil {
			t.Errorf("Fields[ctx] = %v, want nil", v)
		}
	})

	t.Run("string escape sequences", func(t *testing.T) {
		// \" inside a string value
		line := `{"level":"INFO","msg":"say \"hello\""}`
		entry := ParseLine(ctx, logger, []byte(line), "test.log", 0)
		if entry.Malformed {
			t.Fatal("escaped quotes should parse correctly")
		}
		if entry.Message != `say "hello"` {
			t.Errorf("Message = %q, want `say \"hello\"`", entry.Message)
		}
	})

	t.Run("negative integer field", func(t *testing.T) {
		line := `{"level":"INFO","msg":"ok","delta":-99}`
		entry := ParseLine(ctx, logger, []byte(line), "test.log", 0)
		if entry.Malformed {
			t.Fatal("negative int field should not be malformed")
		}
		if v := entry.Fields["delta"]; v != int64(-99) {
			t.Errorf("Fields[delta] = %v, want -99", v)
		}
	})
}

// ── ParseLine – malformed cases ──────────────────────────────────────────────

func TestParseLine_Malformed(t *testing.T) {
	ctx := context.Background()
	logger := newLogger()
	defer logger.Close()

	cases := []struct {
		name  string
		input string
	}{
		{"empty input", ""},
		{"plain text, not JSON", "starting application..."},
		{"missing closing brace", `{"level":"ERROR","msg":"failed"`},
		{"missing colon after key", `{"level" "INFO"}`},
		{"value missing entirely", `{"level":}`},
		{"unterminated string value", `{"msg":"unterminated}`},
		{"unterminated string key", `{unterminated`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := ParseLine(ctx, logger, []byte(tc.input), "test.log", 0)
			if !entry.Malformed {
				t.Errorf("input %q: expected Malformed=true", tc.input)
			}
		})
	}
}

// ── parseTimestamp ───────────────────────────────────────────────────────────

func TestParseTimestamp(t *testing.T) {
	ctx := context.Background()
	logger := newLogger()
	defer logger.Close()

	t.Run("unix seconds (int64)", func(t *testing.T) {
		line := `{"ts":1700000000,"msg":"ok"}`
		entry := ParseLine(ctx, logger, []byte(line), "", 0)
		want := time.Unix(1700000000, 0).UTC()
		if !entry.Timestamp.Equal(want) {
			t.Errorf("got %v, want %v", entry.Timestamp, want)
		}
	})

	t.Run("unix milliseconds (int64 > 1e12)", func(t *testing.T) {
		line := `{"ts":1700000000000,"msg":"ok"}`
		entry := ParseLine(ctx, logger, []byte(line), "", 0)
		want := time.UnixMilli(1700000000000).UTC()
		if !entry.Timestamp.Equal(want) {
			t.Errorf("got %v, want %v", entry.Timestamp, want)
		}
	})

	t.Run("unix seconds as float64", func(t *testing.T) {
		line := `{"ts":1700000000.0,"msg":"ok"}`
		entry := ParseLine(ctx, logger, []byte(line), "", 0)
		want := time.Unix(1700000000, 0).UTC()
		if !entry.Timestamp.Equal(want) {
			t.Errorf("got %v, want %v", entry.Timestamp, want)
		}
	})

	t.Run("RFC3339 string", func(t *testing.T) {
		line := `{"ts":"2024-01-15T10:30:00Z","msg":"ok"}`
		entry := ParseLine(ctx, logger, []byte(line), "", 0)
		want, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
		if !entry.Timestamp.Equal(want.UTC()) {
			t.Errorf("got %v, want %v", entry.Timestamp, want)
		}
	})

	t.Run("RFC3339Nano string", func(t *testing.T) {
		line := `{"ts":"2024-01-15T10:30:00.123456789Z","msg":"ok"}`
		entry := ParseLine(ctx, logger, []byte(line), "", 0)
		if entry.Timestamp.IsZero() {
			t.Error("RFC3339Nano timestamp should not be zero")
		}
		if entry.Timestamp.Nanosecond() != 123456789 {
			t.Errorf("nanoseconds = %d, want 123456789", entry.Timestamp.Nanosecond())
		}
	})

	t.Run("Kitchen string anchors to today", func(t *testing.T) {
		line := `{"ts":"3:04PM","msg":"ok"}`
		entry := ParseLine(ctx, logger, []byte(line), "", 0)
		now := time.Now().UTC()
		if entry.Timestamp.IsZero() {
			t.Fatal("Kitchen timestamp should not be zero")
		}
		if entry.Timestamp.Year() != now.Year() {
			t.Errorf("Kitchen year = %d, want %d", entry.Timestamp.Year(), now.Year())
		}
	})

	t.Run("unrecognised string gives zero time", func(t *testing.T) {
		line := `{"ts":"not-a-time","msg":"ok"}`
		entry := ParseLine(ctx, logger, []byte(line), "", 0)
		if !entry.Timestamp.IsZero() {
			t.Errorf("expected zero time, got %v", entry.Timestamp)
		}
	})
}

// ── ParseTimeParam ───────────────────────────────────────────────────────────

func TestParseTimeParam(t *testing.T) {
	t.Run("unix seconds string", func(t *testing.T) {
		got, err := ParseTimeParam("1700000000")
		if err != nil {
			t.Fatal(err)
		}
		want := time.Unix(1700000000, 0).UTC()
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("unix milliseconds string", func(t *testing.T) {
		got, err := ParseTimeParam("1700000000000")
		if err != nil {
			t.Fatal(err)
		}
		want := time.UnixMilli(1700000000000).UTC()
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("RFC3339", func(t *testing.T) {
		got, err := ParseTimeParam("2024-01-15T10:30:00Z")
		if err != nil {
			t.Fatal(err)
		}
		want, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
		if !got.Equal(want.UTC()) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("HTML5 datetime-local with seconds", func(t *testing.T) {
		got, err := ParseTimeParam("2024-01-15T10:30:45")
		if err != nil {
			t.Fatal(err)
		}
		if got.Hour() != 10 || got.Minute() != 30 || got.Second() != 45 {
			t.Errorf("unexpected time components: %v", got)
		}
	})

	t.Run("HTML5 datetime-local without seconds", func(t *testing.T) {
		got, err := ParseTimeParam("2024-01-15T10:30")
		if err != nil {
			t.Fatal(err)
		}
		if got.Hour() != 10 || got.Minute() != 30 {
			t.Errorf("unexpected time components: %v", got)
		}
	})

	t.Run("unrecognised format returns error", func(t *testing.T) {
		_, err := ParseTimeParam("yesterday")
		if err == nil {
			t.Error("expected an error for unrecognised format")
		}
	})
}
