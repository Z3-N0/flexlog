package flexlog

import "github.com/Z3-N0/flexlog/core"

type Logger = core.Logger
type Level = core.Level
type Option = core.Option

const (
	LevelTrace    = core.LevelTrace
	LevelDebug    = core.LevelDebug
	LevelInfo     = core.LevelInfo
	LevelWarn     = core.LevelWarn
	LevelError    = core.LevelError
	LevelFatal    = core.LevelFatal
	LevelDisabled = core.LevelDisabled
)

var New = core.New
var WithLevel = core.WithLevel
var WithTraceID = core.WithTraceID
var TraceIDFromContext = core.TraceIDFromContext
