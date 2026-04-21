package server

import (
	"context"
	"testing"

	"github.com/Z3-N0/flexlog"
)

func TestParseLine(t *testing.T) {
	logger := flexlog.New(flexlog.WithLevel(flexlog.LevelTrace))
	defer logger.Close()
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		wantMsg  string
		wantLvl  string
		wantTrce string
		isMal    bool
	}{
		{
			name:     "Standard valid log",
			input:    `{"level":"INFO","ts":1776784434001,"msg":"indexing complete","trace_id":"abc-123"}`,
			wantMsg:  "indexing complete",
			wantLvl:  "INFO",
			wantTrce: "abc-123",
			isMal:    false,
		},
		{
			name:    "Malformed: missing closing brace",
			input:   `{"level":"ERROR","msg":"failed"`,
			isMal:   true,
		},
		{
			name:    "Malformed: not JSON",
			input:   `starting application...`,
			isMal:   true,
		},
		{
			name:    "Extra whitespace and nested-like fields",
			input:   `  {  "level" : "DEBUG" , "msg" : "hello world" }  `,
			wantMsg: "hello world",
			wantLvl: "DEBUG",
			isMal:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(ctx, logger, []byte(tt.input), "test.log", 0)
			if entry.Malformed != tt.isMal {
				t.Errorf("Malformed = %v, want %v", entry.Malformed, tt.isMal)
			}
			if !tt.isMal {
				if entry.Message != tt.wantMsg {
					t.Errorf("Message = %q, want %q", entry.Message, tt.wantMsg)
				}
				if entry.Level != tt.wantLvl {
					t.Errorf("Level = %q, want %q", entry.Level, tt.wantLvl)
				}
				if tt.wantTrce != "" && entry.TraceID != tt.wantTrce {
					t.Errorf("TraceID = %q, want %q", entry.TraceID, tt.wantTrce)
				}
			}
		})
	}
}
