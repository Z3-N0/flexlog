package formatter

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

// --- JSON Structure Tests ---

func TestFormatBasic(t *testing.T) {
	ts := int64(1672531200000) // Fixed timestamp for consistency
	level := "INFO"
	traceID := "trace-123"
	msg := "hello world"
	fields := map[string]any{
		"user_id": 42,
		"active":  true,
	}

	got, err := Format(level, ts, traceID, msg, fields)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Verify it is valid JSON by unmarshaling it
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Errorf("Format produced invalid JSON: %v. Output: %s", err, string(got))
	}

	// Check fixed fields
	if parsed["level"] != level {
		t.Errorf("level = %v, want %v", parsed["level"], level)
	}
	if parsed["ts"] != float64(1672531200000) { // JSON numbers are float64 in Go maps
		t.Errorf("ts = %v, want %v", parsed["ts"], 1672531200000)
	}
	if parsed["trace_id"] != traceID {
		t.Errorf("trace_id = %v, want %v", parsed["trace_id"], traceID)
	}
	if parsed["msg"] != msg {
		t.Errorf("msg = %v, want %v", parsed["msg"], msg)
	}
	if parsed["user_id"] != float64(42) {
		t.Errorf("field user_id = %v, want 42", parsed["user_id"])
	}
}

// --- Escaping and Edge Case Tests ---

func TestFormatEscaping(t *testing.T) {
	msgWithQuotes := `message with "quotes" and \backslashes\ and
newline`

	got, err := Format("INFO", time.Now(), "", msgWithQuotes, nil)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// If escaping is wrong, json.Unmarshal will fail
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("Escaping failed, produced invalid JSON: %s", string(got))
	}

	if parsed["msg"] != msgWithQuotes {
		t.Errorf("Escaped message mismatch. \nGot: %v\nWant: %v", parsed["msg"], msgWithQuotes)
	}
}

func TestFormatKeyEscaping(t *testing.T) {
	fields := map[string]any{`key"with"quotes`: "val"}
	got, _ := Format("INFO", time.Now(), "", "msg", fields)
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Errorf("Key with quotes produced invalid JSON: %s", got)
	}
}

func TestFormatEmptyFields(t *testing.T) {
	// Ensure that when fields is nil or empty, we don't have a trailing comma error
	got, err := Format("INFO", time.Now(), "id", "msg", nil)
	if err != nil {
		t.Fatalf("Format failed with nil fields: %v", err)
	}

	if got[len(got)-1] != '}' {
		t.Errorf("JSON did not end with brace: %s", string(got))
	}

	if bytes.Contains(got, []byte(",}")) {
		t.Errorf("JSON contains invalid trailing comma: %s", string(got))
	}
}

func TestFormatTypes(t *testing.T) {
	fields := map[string]any{
		"string":  "val",
		"int":     int(10),
		"int64":   int64(20),
		"float":   3.14,
		"bool":    false,
		"string2": "another val",
		"custom":  []int{1, 2}, // Should hit stringify
	}

	got, err := Format("DEBUG", time.Now(), "", "test", fields)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var parsed map[string]any
	err = json.Unmarshal(got, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	// Verify the custom type was stringified correctly
	if parsed["custom"] != "[1 2]" {
		t.Errorf("Expected custom type to be stringified to '[1 2]', got %v", parsed["custom"])
	}
}

// --- Performance principles: Benchmarking ---

func BenchmarkFormat(b *testing.B) {
	ts := time.Now()
	fields := map[string]any{
		"id":      "12345",
		"attempt": 3,
		"success": true,
		"ratio":   0.95,
	}

	b.ReportAllocs() // This verifies your "zero-allocation" goal (except the final result slice)

	for b.Loop() {
		_, _ = Format("INFO", ts, "trace-abc", "log message", fields)
	}
}

func TestFormatControlCharacters(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"null byte", "msg\x00end"},
		{"bell", "msg\x07end"},
		{"form feed", "msg\x0Cend"},
		{"vertical tab", "msg\x0Bend"},
		{"unit separator", "msg\x1Fend"},
		{"tab", "msg\tend"},
		{"carriage return", "msg\rend"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Format("INFO", int64(0), "", tc.msg, nil)
			if err != nil {
				t.Fatalf("Format failed: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(got, &parsed); err != nil {
				t.Errorf("control char %q produced invalid JSON: %s", tc.msg, got)
			}
			if parsed["msg"] != tc.msg {
				t.Errorf("msg roundtrip failed.\ngot:  %v\nwant: %v", parsed["msg"], tc.msg)
			}
		})
	}
}

func TestFormatControlCharactersInKeys(t *testing.T) {
	fields := map[string]any{
		"key\x00null": "value",
		"key\ttab":    "value",
	}
	got, err := Format("INFO", int64(0), "", "msg", fields)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Errorf("control chars in keys produced invalid JSON: %s", got)
	}
}

func TestFormatControlCharactersInTraceID(t *testing.T) {
	got, err := Format("INFO", int64(0), "trace\x01id", "msg", nil)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Errorf("control char in traceID produced invalid JSON: %s", got)
	}
}
