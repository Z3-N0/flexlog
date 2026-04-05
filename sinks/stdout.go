package sinks

import (
	"os"

	"github.com/Z3-N0/flexlog/formatter"
)

// StdoutSink writes formatted log entries to stdout.
type StdoutSink struct{}

// Stdout is the default sink
var Stdout = &StdoutSink{}

func (s *StdoutSink) Write(level string, ts any, traceID string, msg string, fields map[string]any) error {
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
	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}
	if _, err := os.Stdout.Write([]byte{'\n'}); err != nil {
		return err
	}
	return nil
}

func (s *StdoutSink) Close() error {
	return nil // stdout is never closed
}
