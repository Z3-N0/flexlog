# flexlog

A lightweight, structured, leveled logger for Go. Built for low overhead
no reflection, no unnecessary allocations, just fast JSON output to stdout.

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

    log.Info(ctx, "server started", "port", 8080)
}
```

Output:

```json
{ "level": "INFO", "ts": 1775315926040, "msg": "server started", "port": 8080 }
```

## Levels

flexlog supports six log levels, in ascending order of severity:

```go
log.Trace(ctx, "very detailed info")
log.Debug(ctx, "useful during development")
log.Info(ctx, "normal operational events")
log.Warn(ctx, "something unexpected, but recovered")
log.Error(ctx, "failure that needs attention")
log.Fatal(ctx, "unrecoverable failure") // logs then calls os.Exit(1)
```

The default minimum level is `Info`. Entries below the minimum are silently dropped.

## Options

### Set minimum level

```go
log := flexlog.New(flexlog.WithLevel(flexlog.LevelDebug))
```

### Attach persistent fields

Use `With` to create a child logger with fields that appear on every entry:

```go
log := flexlog.New()
serviceLog := log.With("service", "auth", "env", "prod")

serviceLog.Info(ctx, "request received", "user_id", 42)
// {"level":"INFO","ts":...,"msg":"request received","service":"auth","env":"prod","user_id":42}
```

Child loggers inherit their parent's level and fields. The parent is unchanged.

## Distributed Tracing

Attach a trace ID to a context once and it flows through automatically:

```go
ctx = flexlog.WithTraceID(ctx, "abc-123")
log.Info(ctx, "processing request")
// {"level":"INFO","ts":...,"trace_id":"abc-123","msg":"processing request"}
```

Pull the trace ID back out with `flexlog.TraceIDFromContext(ctx)` if needed.

## Output Format

All entries are written as newline-delimited JSON to stdout. Field order is always:
level → ts → trace_id (omitted if empty) → msg → your fields

Timestamps are Unix milliseconds (`ts`).

## Roadmap

- **v1** - JSON output to stdout ✅
- **v1.5** - Pluggable sinks (file, stderr, remote), configurable timestamp format via `WithTimeFormat`
- **v2** - Web-based log viewer, searchable and sortable, shipped as a single binary

## License

Apache 2.0
