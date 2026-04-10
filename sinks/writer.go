package sinks

import (
	"io"
	"sync"

	"github.com/Z3-N0/flexlog/formatter"
)

// WriterSink writes formatted log entries to any io.Writer.
// This is the generic sink which can be wrapped for any destination
type WriterSink struct {
	mu sync.Mutex
	w  io.Writer
}

// NewWriterSink creates a sink that writes to the given io.Writer.
func NewWriterSink(w io.Writer) *WriterSink {
	return &WriterSink{w: w}
}

func (s *WriterSink) Write(level string, ts any, traceID string, msg string, fields map[string]any) error {
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
	if _, err := s.w.Write(out); err != nil {
		return err
	}
	return nil
}

func (s *WriterSink) Close() error {
	// If the writer is also a Closer, close it
	if c, ok := s.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
