package server

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Z3-N0/flexlog"
)

// Query holds all filter and pagination parameters for a log search.
type Query struct {
	Search        string
	Levels        []string
	From          time.Time
	To            time.Time
	TraceID       string
	Files         []string
	Page          int
	PageSize      int
	SortDesc      bool
	ShowMalformed bool
}

// QueryResult holds the matched entries and pagination metadata.
type QueryResult struct {
	Entries    []LogEntry
	TotalMatch int
	Page       int
	TotalPages int
}

// Execute runs the query across all indexed files concurrently and returns a paginated result.
func Execute(ctx context.Context, logger *flexlog.Logger, q Query, indexes map[string]*FileIndex) QueryResult {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 50
	}

	targetFiles := resolveFiles(q, indexes)
	if len(targetFiles) == 0 {
		return QueryResult{Page: 1, TotalPages: 1}
	}

	// one goroutine per file
	var mu sync.Mutex
	var all []LogEntry
	var wg sync.WaitGroup

	for _, idx := range targetFiles {
		wg.Add(1)
		go func(idx *FileIndex) {
			defer wg.Done()
			matches := scanFile(ctx, logger, idx, q)
			mu.Lock()
			all = append(all, matches...)
			mu.Unlock()
		}(idx)
	}
	wg.Wait()

	// sort by timestamp
	sort.Slice(all, func(i, j int) bool {
		if q.SortDesc {
			return all[i].Timestamp.After(all[j].Timestamp)
		}
		return all[i].Timestamp.Before(all[j].Timestamp)
	})

	total := len(all)
	totalPages := (total + q.PageSize - 1) / q.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// paginate
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}

	return QueryResult{
		Entries:    all[start:end],
		TotalMatch: total,
		Page:       q.Page,
		TotalPages: totalPages,
	}
}

// scanFile scans a single indexed file and returns all matching entries.
func scanFile(ctx context.Context, logger *flexlog.Logger, idx *FileIndex, q Query) []LogEntry {
	f, err := os.Open(idx.Path)
	if err != nil {
		return nil
	}
	defer f.Close()

	reader := bufio.NewReaderSize(f, 256*1024)
	searchTerm := bytes.ToLower([]byte(q.Search))
	levelSet := toLevelSet(q.Levels)

	var matches []LogEntry

	for _, offset := range idx.Offsets {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			continue
		}
		reader.Reset(f)

		line, err := reader.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			continue
		}
		line = bytes.TrimRight(line, "\n\r")

		// substring filter on raw line
		if len(searchTerm) > 0 && !bytes.Contains(bytes.ToLower(line), searchTerm) {
			continue
		}

		entry := ParseLine(ctx, logger, line, idx.Path, offset)

		// malformed filter
		if entry.Malformed && !q.ShowMalformed {
			continue
		}

		// level filter
		if len(levelSet) > 0 && !entry.Malformed {
			if _, ok := levelSet[entry.Level]; !ok {
				continue
			}
		}

		// trace ID filter
		if q.TraceID != "" && entry.TraceID != q.TraceID {
			continue
		}

		// time range filter
		if !q.From.IsZero() && entry.Timestamp.Before(q.From) {
			continue
		}
		if !q.To.IsZero() && entry.Timestamp.After(q.To) {
			continue
		}

		matches = append(matches, entry)
	}

	return matches
}

// resolveFiles returns the FileIndex entries the query should run against.
// Empty q.Files means all indexed files.
func resolveFiles(q Query, indexes map[string]*FileIndex) []*FileIndex {
	if len(q.Files) == 0 {
		return nil
	}
	result := make([]*FileIndex, 0, len(q.Files))
	for _, f := range q.Files {
		if idx, ok := indexes[f]; ok {
			result = append(result, idx)
		}
	}
	return result
}

// toLevelSet converts a slice of level strings to a set for O(1) lookup.
func toLevelSet(levels []string) map[string]struct{} {
	if len(levels) == 0 {
		return nil
	}

	set := make(map[string]struct{})
	for _, l := range levels {
		if l == "" {
			continue
		}

		parts := strings.Split(l, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				// Ensure case matching with ParseLine (usually Uppercase)
				set[strings.ToUpper(trimmed)] = struct{}{}
			}
		}
	}

	if len(set) == 0 {
		return nil
	}
	return set
}
