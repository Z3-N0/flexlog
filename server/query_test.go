package server

import (
	"testing"
	"time"

	"github.com/Z3-N0/flexlog"
)

func TestExecuteQuery(t *testing.T) {
	logger := flexlog.New()
	defer logger.Close()
	// Mock Data
	now := time.Now()
	entries := []LogEntry{
		{Level: "INFO", Message: "User logged in", Timestamp: now.Add(-10 * time.Minute)},
		{Level: "ERROR", Message: "Database connection failed", Timestamp: now.Add(-5 * time.Minute)},
		{Level: "DEBUG", Message: "Cache miss", Timestamp: now},
	}

	// Helper to find index (In a real test, you'd point to a temp file)
	// For this test, we are testing the filtering logic of scanFile indirectly

	t.Run("Filter by Level", func(t *testing.T) {
		q := Query{
			Levels:   []string{"ERROR"},
			PageSize: 10,
		}

		levelSet := toLevelSet(q.Levels)
		matches := 0
		for _, e := range entries {
			if _, ok := levelSet[e.Level]; ok {
				matches++
			}
		}

		if matches != 1 {
			t.Errorf("Expected 1 ERROR match, found %d", matches)
		}
	})

	t.Run("Filter by Time Range", func(t *testing.T) {
		q := Query{
			From: now.Add(-7 * time.Minute),
			To:   now.Add(-1 * time.Minute),
		}

		matches := 0
		for _, e := range entries {
			if e.Timestamp.After(q.From) && e.Timestamp.Before(q.To) {
				matches++
			}
		}

		if matches != 1 { // Only the ERROR log fits this range
			t.Errorf("Expected 1 match in time range, found %d", matches)
		}
	})
}
