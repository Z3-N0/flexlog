package sinks

// Sink is the interface that all log destinations must implement.

type Sink interface {
	Write(level string, ts any, traceID string, msg string, fields map[string]any) error // receives all fields of an entry, formats it and writes to destination
	Close() error                                                                        // call when done with sink to flush buffer and free resources
}
