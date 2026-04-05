package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/Z3-N0/flexlog"
)

// example SQLiteSink to demonstrate how to implement a custom sink, any type with Write and Close satisfies the sink interface.
type SQLiteSink struct {
	db *sql.DB
}

func (s *SQLiteSink) Write(level string, ts any, traceID string, msg string, fields map[string]any) error {
	_, err := s.db.Exec(
		`INSERT INTO logs (level, ts, trace_id, msg) VALUES (?, ?, ?, ?)`,
		level, ts, traceID, msg,
	)
	return err
}

func (s *SQLiteSink) Close() error {
	return s.db.Close()
}

func main() {
	// file sink
	fileSink, err := flexlog.NewFileSink("app.log")
	if err != nil {
		log.Fatalf("failed to create file sink: %v", err)
	}

	// writer sink - any io.Writer works
	var buf bytes.Buffer
	writerSink := flexlog.NewWriterSink(&buf)

	logger := flexlog.New(
		flexlog.WithTimeFormat(flexlog.TimeKitchen),
		flexlog.WithSink(flexlog.Stdout),
		flexlog.WithSink(fileSink),
		flexlog.WithSink(writerSink),
		// flexlog.WithSink(&SQLiteSink{db: db}), this is how the actual writer sink would fit in
	)
	defer logger.Close()

	ctx := context.Background()

	// basic logging
	logger.Info(ctx, "server started", "port", 8080)

	// child logger with persistent fields
	reqLog := logger.With("service", "auth", "env", "prod")
	reqLog.Info(ctx, "request received", "user_id", 42)

	// trace ID flows through context automatically
	ctx = flexlog.WithTraceID(ctx, "abc-123")
	logger.Info(ctx, "processing request")
	logger.Warn(ctx, "high latency detected", "latency_ms", 320)
	logger.Error(ctx, "database timeout", "retry", true)

	// show what the writer sink captured
	fmt.Println("WriterSink captured:", buf.String())
}
