package sinks_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Z3-N0/flexlog/sinks"
)

// helper to build standard args for Write calls
func writeArgs() (string, any, string, string, map[string]any) {
	return "INFO", time.Now().UnixMilli(), "trace-123", "test message", map[string]any{"key": "val"}
}

// helper to parse and validate JSON output
func mustParseJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(data))
	}
	return parsed
}

// errWriter always fails on Write
type errWriter struct{}

func (e *errWriter) Write(p []byte) (int, error) {
	return 0, errors.New("write failed")
}

// --- StdoutSink ---

func TestStdoutSinkWrite(t *testing.T) {
	err := sinks.Stdout.Write(writeArgs())
	if err != nil {
		t.Errorf("StdoutSink.Write returned error: %v", err)
	}
}

func TestStdoutSinkClose(t *testing.T) {
	if err := sinks.Stdout.Close(); err != nil {
		t.Errorf("StdoutSink.Close returned error: %v", err)
	}
}

// --- StderrSink ---

func TestStderrSinkWrite(t *testing.T) {
	err := sinks.Stderr.Write(writeArgs())
	if err != nil {
		t.Errorf("StderrSink.Write returned error: %v", err)
	}
}

func TestStderrSinkClose(t *testing.T) {
	if err := sinks.Stderr.Close(); err != nil {
		t.Errorf("StderrSink.Close returned error: %v", err)
	}
}

// --- WriterSink ---

func TestWriterSinkWrite(t *testing.T) {
	var buf bytes.Buffer
	s := sinks.NewWriterSink(&buf)

	if err := s.Write(writeArgs()); err != nil {
		t.Fatalf("WriterSink.Write returned error: %v", err)
	}

	mustParseJSON(t, bytes.TrimRight(buf.Bytes(), "\n"))
}

func TestWriterSinkClose(t *testing.T) {
	var buf bytes.Buffer
	s := sinks.NewWriterSink(&buf)
	if err := s.Close(); err != nil {
		t.Errorf("WriterSink.Close returned error: %v", err)
	}
}

func TestWriterSinkClosesUnderlying(t *testing.T) {
	f, err := os.CreateTemp("", "flexlog-writer-test-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	s := sinks.NewWriterSink(f)
	if err := s.Close(); err != nil {
		t.Errorf("WriterSink.Close on file returned error: %v", err)
	}
}

func TestWriterSinkConcurrent(t *testing.T) {
	var buf bytes.Buffer
	s := sinks.NewWriterSink(&buf)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Write(writeArgs()); err != nil {
				t.Errorf("concurrent Write returned error: %v", err)
			}
		}()
	}
	wg.Wait()

	// each write should produce a valid JSON line
	lines := bytes.Split(bytes.TrimRight(buf.Bytes(), "\n"), []byte("\n"))
	if len(lines) != 100 {
		t.Errorf("expected 100 lines, got %d", len(lines))
	}
	for _, line := range lines {
		mustParseJSON(t, line)
	}
}

func TestWriterSinkWriteError(t *testing.T) {
	s := sinks.NewWriterSink(&errWriter{})
	err := s.Write(writeArgs())
	if err == nil {
		t.Error("expected error from failing writer, got nil")
	}
}

func TestWriterSinkEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	s := sinks.NewWriterSink(&buf)

	if err := s.Write("INFO", time.Now().UnixMilli(), "", "msg", nil); err != nil {
		t.Fatalf("Write with nil fields returned error: %v", err)
	}

	mustParseJSON(t, bytes.TrimRight(buf.Bytes(), "\n"))
}

func TestWriterSinkEmptyTraceID(t *testing.T) {
	var buf bytes.Buffer
	s := sinks.NewWriterSink(&buf)

	if err := s.Write("INFO", time.Now().UnixMilli(), "", "msg", nil); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	parsed := mustParseJSON(t, bytes.TrimRight(buf.Bytes(), "\n"))
	if _, ok := parsed["trace_id"]; ok {
		t.Error("trace_id should be omitted when empty")
	}
}

func TestWriterSinkAllFieldTypes(t *testing.T) {
	var buf bytes.Buffer
	s := sinks.NewWriterSink(&buf)

	fields := map[string]any{
		"string":  "val",
		"int":     int(10),
		"int64":   int64(20),
		"float":   3.14,
		"bool":    false,
		"custom":  []int{1, 2},
	}

	if err := s.Write("DEBUG", time.Now().UnixMilli(), "", "test", fields); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	mustParseJSON(t, bytes.TrimRight(buf.Bytes(), "\n"))
}

func TestWriterSinkAllLevels(t *testing.T) {
	levels := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	for _, level := range levels {
		var buf bytes.Buffer
		s := sinks.NewWriterSink(&buf)

		if err := s.Write(level, time.Now().UnixMilli(), "", "msg", nil); err != nil {
			t.Errorf("Write with level %s returned error: %v", level, err)
		}

		parsed := mustParseJSON(t, bytes.TrimRight(buf.Bytes(), "\n"))
		if parsed["level"] != level {
			t.Errorf("expected level %s, got %v", level, parsed["level"])
		}
	}
}

func TestWriterSinkRFC3339Timestamp(t *testing.T) {
	var buf bytes.Buffer
	s := sinks.NewWriterSink(&buf)

	ts := time.Now().UTC().Format(time.RFC3339)
	if err := s.Write("INFO", ts, "", "msg", nil); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	parsed := mustParseJSON(t, bytes.TrimRight(buf.Bytes(), "\n"))
	if parsed["ts"] != ts {
		t.Errorf("expected ts %s, got %v", ts, parsed["ts"])
	}
}

// --- FileSink ---

func TestFileSinkWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	s, err := sinks.NewFileSink(path)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("FileSink.Close returned error: %v", err)
		}
	}()

	if err := s.Write(writeArgs()); err != nil {
		t.Fatalf("FileSink.Write returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read log file: %v", err)
	}

	mustParseJSON(t, bytes.TrimRight(data, "\n"))
}

func TestFileSinkAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	s, err := sinks.NewFileSink(path)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	if err := s.Write(writeArgs()); err != nil {
		t.Fatalf("FileSink.Write returned error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("FileSink.Close returned error: %v", err)
	}

	s, err = sinks.NewFileSink(path)
	if err != nil {
		t.Fatalf("NewFileSink failed on second open: %v", err)
	}
	if err := s.Write(writeArgs()); err != nil {
		t.Fatalf("FileSink.Write returned error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("FileSink.Close returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read log file: %v", err)
	}
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
	if len(lines) != 2 {
		t.Errorf("expected 2 log lines, got %d", len(lines))
	}
	for _, line := range lines {
		mustParseJSON(t, line)
	}
}

func TestFileSinkInvalidPath(t *testing.T) {
	_, err := sinks.NewFileSink("/nonexistent/path/test.log")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestFileSinkClose(t *testing.T) {
	dir := t.TempDir()
	s, err := sinks.NewFileSink(filepath.Join(dir, "test.log"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("FileSink.Close returned error: %v", err)
	}
}

func TestFileSinkConcurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.log")

	s, err := sinks.NewFileSink(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("FileSink.Close returned error: %v", err)
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Write(writeArgs()); err != nil {
				t.Errorf("concurrent Write returned error: %v", err)
			}
		}()
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read log file: %v", err)
	}
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
	if len(lines) != 100 {
		t.Errorf("expected 100 lines, got %d", len(lines))
	}
	for _, line := range lines {
		mustParseJSON(t, line)
	}
}

func TestFileSinkEmptyFields(t *testing.T) {
	dir := t.TempDir()
	s, err := sinks.NewFileSink(filepath.Join(dir, "test.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("FileSink.Close returned error: %v", err)
		}
	}()

	if err := s.Write("INFO", time.Now().UnixMilli(), "", "msg", nil); err != nil {
		t.Fatalf("Write with nil fields returned error: %v", err)
	}
}

func TestFileSinkEmptyTraceID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	s, err := sinks.NewFileSink(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("FileSink.Close returned error: %v", err)
		}
	}()

	if err := s.Write("INFO", time.Now().UnixMilli(), "", "msg", nil); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	data, _ := os.ReadFile(path)
	parsed := mustParseJSON(t, bytes.TrimRight(data, "\n"))
	if _, ok := parsed["trace_id"]; ok {
		t.Error("trace_id should be omitted when empty")
	}
}
