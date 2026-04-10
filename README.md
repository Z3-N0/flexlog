# Flexlog

Fast, structured and customisable logging for Go. No reflection, no bloat - just clean JSON output, even to multiple sinks.

## Installation

```bash
go get github.com/Z3-N0/flexlog
```

Requires Go 1.25+.

## Quick Start

```go
import (
    "context"
    "github.com/Z3-N0/flexlog"
)

func main() {
    ctx := context.Background()
    log := flexlog.New()
    defer log.Close()

    log.Info(ctx, "server started", "port", 8080)
}
```

Output:

```json
{ "level": "INFO", "ts": 1775315926040, "msg": "server started", "port": 8080 }
```

## Levels

```go
log.Trace(ctx, "very detailed info")
log.Debug(ctx, "useful during development")
log.Info(ctx, "normal operational events")
log.Warn(ctx, "something unexpected, but recovered")
log.Error(ctx, "failure that needs attention")
log.Fatal(ctx, "unrecoverable failure") // logs then calls os.Exit(1) by default
```

The default minimum level is `Info`. Entries below the minimum are silently dropped.

```go
log := flexlog.New(flexlog.WithLevel(flexlog.LevelDebug))
```

## Sinks

Sinks are where your logs go. You can attach multiple.

```go
fileSink, err := flexlog.NewFileSink("app.log")
if err != nil {
    log.Fatal(err)
}

logger := flexlog.New(
    flexlog.WithSink(flexlog.Stdout),
    flexlog.WithSink(fileSink),
)
defer logger.Close()
```

### Built-in sinks

| Sink                        | Description                          |
| --------------------------- | ------------------------------------ |
| `flexlog.Stdout`            | Writes to stdout (default)           |
| `flexlog.Stderr`            | Writes to stderr                     |
| `flexlog.NewFileSink(path)` | Appends to a file, creates if needed |
| `flexlog.NewWriterSink(w)`  | Wraps any `io.Writer`                |

### Custom sinks

Any type with `Write` and `Close` satisfies the `Sink` interface:

```go
type Sink interface {
    Write(level string, ts any, traceID string, msg string, fields map[string]any) error
    Close() error
}
```

Example - writing to a database:

```go
type SQLiteSink struct{ db *sql.DB }

func (s *SQLiteSink) Write(level string, ts any, traceID string, msg string, fields map[string]any) error {
    _, err := s.db.Exec(`INSERT INTO logs (level, ts, msg) VALUES (?, ?, ?)`, level, ts, msg)
    return err
}

func (s *SQLiteSink) Close() error { return s.db.Close() }
```

```go
logger := flexlog.New(flexlog.WithSink(&SQLiteSink{db: db}))
```

## Timestamp Formats

```go
logger := flexlog.New(flexlog.WithTimeFormat(flexlog.TimeRFC3339))
```

| Constant          | Output                             |
| ----------------- | ---------------------------------- |
| `TimeUnixMilli`   | `1775315926040` (default)          |
| `TimeUnixSec`     | `1775315926`                       |
| `TimeRFC3339`     | `"2026-04-04T10:30:00Z"`           |
| `TimeRFC3339Nano` | `"2026-04-04T10:30:00.000000000Z"` |
| `TimeKitchen`     | `"3:04PM"`                         |

## Persistent Fields

`With` returns a child logger with fields attached to every entry. If no
arguments are passed, the same logger is returned unchanged:

```go
child := logger.With() // returns logger itself, no allocation
```

Call-site fields take precedence over persistent fields when keys collide:

```go
log := logger.With("env", "prod")
log.Info(ctx, "msg", "env", "staging")
// env will be "staging" in the output
```

Keys and values must alternate. If an odd number of arguments is passed,
the final key is logged with the value `"MISSING"` as a signal that
something is wrong at the call site:

```go
log.Info(ctx, "msg", "orphaned_key")
// output will contain: "orphaned_key": "MISSING"
```

## Distributed Tracing

Attach a trace ID to a context once and it flows through automatically:

```go
ctx = flexlog.WithTraceID(ctx, "abc-123")
logger.Info(ctx, "processing request")
// {"level":"INFO","ts":...,"trace_id":"abc-123","msg":"processing request"}
```

## Fatal Hook

Control what happens after a `Fatal` log:

```go
// default - logs and exits
logger := flexlog.New(flexlog.WithFatalHook(flexlog.FatalHookExit))

// logs but does nothing - useful in tests
logger := flexlog.New(flexlog.WithFatalHook(flexlog.FatalHookNoop))

// logs then panics
logger := flexlog.New(flexlog.WithFatalHook(flexlog.FatalHookPanic))
```

## Closing the Logger

Always defer `Close()` after creating a logger. Once closed, any further
log calls are silently dropped — no writes to closed sinks, no panics:

```go
log := flexlog.New()
defer log.Close()
```

## Roadmap

- **v1** - JSON output to stdout ✅
- **v1.5** - Pluggable sinks, configurable timestamp format, fatal hook ✅
- **v2** - Web-based log viewer, searchable and sortable, shipped as a single binary

## License

Apache 2.0
