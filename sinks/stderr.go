package sinks

import (
	"os"

	"github.com/Z3-N0/flexlog/formatter"
)

// StderrSink writes formatted log entries to stderr.
type StderrSink struct{}

var Stderr = &StderrSink{}

func (s *StderrSink) Write(level string, ts any, traceID string, msg string, fields map[string]any) error {
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
	out = append(out, '\n')
	if _, err := os.Stderr.Write(out); err != nil {
		return err
	}
	return nil
}

func (s *StderrSink) Close() error {
	return nil // stderr is never closed
}
