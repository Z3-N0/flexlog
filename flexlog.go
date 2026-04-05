package flexlog

import (
	"github.com/Z3-N0/flexlog/core"
	"github.com/Z3-N0/flexlog/sinks"
)

// type aliases
type Logger = core.Logger
type Level = core.Level
type Option = core.Option
type TimeFormat = core.TimeFormat

// levels
const (
	LevelTrace    = core.LevelTrace
	LevelDebug    = core.LevelDebug
	LevelInfo     = core.LevelInfo
	LevelWarn     = core.LevelWarn
	LevelError    = core.LevelError
	LevelFatal    = core.LevelFatal
	LevelDisabled = core.LevelDisabled
)

// time formats
const (
	TimeUnixMilli   = core.TimeUnixMilli
	TimeUnixSec     = core.TimeUnixSec
	TimeRFC3339     = core.TimeRFC3339
	TimeRFC3339Nano = core.TimeRFC3339Nano
	TimeKitchen     = core.TimeKitchen
)

// logger construction
var New = core.New
var WithLevel = core.WithLevel
var WithSink = core.WithSink
var WithTimeFormat = core.WithTimeFormat

// trace IDs
var WithTraceID = core.WithTraceID
var TraceIDFromContext = core.TraceIDFromContext

// sinks
var Stdout = sinks.Stdout
var Stderr = sinks.Stderr
var NewFileSink = sinks.NewFileSink
var NewWriterSink = sinks.NewWriterSink
