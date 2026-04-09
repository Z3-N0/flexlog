package sinks

import (
	"os"
	"sync"

	"github.com/Z3-N0/flexlog/formatter"
)

// FileSink writes formatted log entries to a file.
// The file is created if it doesn't exist, and appended to if it does.
type FileSink struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileSink opens or creates a log file at the given path.
func NewFileSink(path string) (*FileSink, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileSink{file: f}, nil
}

func (s *FileSink) Write(level string, ts any, traceID string, msg string, fields map[string]any) error {
	out, err := formatter.Format(
		level,
		ts,
		traceID,
		msg,
		fields,
	)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	out = append(out, '\n')
	if _, err := s.file.Write(out); err != nil {
		return err
	}
	return nil
}

// Close flushes and closes file. Always call when done with logger.
func (s *FileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.file.Close()
}
