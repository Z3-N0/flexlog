package core

// FatalHook controls what happens after a Fatal log entry is written.
type FatalHook int8

const (
	FatalHookExit  FatalHook = iota // default — logs and calls os.Exit(1)
	FatalHookNoop                   // logs but does nothing, useful in tests
	FatalHookPanic                  // logs then panics
)
