package server

import (
	"bytes"
	"sort"
	"sync"
	"time"
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
func Execute(q Query, indexes map[string]*FileIndex) QueryResult {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 50
	}

	targetFiles := resolveFiles(q, indexes)

	// fan out — one goroutine per file
	var mu sync.Mutex
	var all []LogEntry
	var wg sync.WaitGroup

	for _, idx := range targetFiles {
		wg.Add(1)
		go func(idx *FileIndex) {
			defer wg.Done()
			matches := scanFile(idx, q)
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
func scanFile(idx *FileIndex, q Query) []LogEntry {
	searchTerm := bytes.ToLower([]byte(q.Search))
	levelSet := toLevelSet(q.Levels)

	var matches []LogEntry

	for i, offset := range idx.Offsets {
		line, err := ReadLine(idx.Path, offset)
		if err != nil {
			continue
		}

		// substring filter on raw line — cheap, done first
		if len(searchTerm) > 0 && !bytes.Contains(bytes.ToLower(line), searchTerm) {
			continue
		}

		entry := ParseLine(line, idx.Path, offset)
		_ = i

		// malformed filter
		if entry.Malformed && !q.ShowMalformed {
			continue
		}

		// level filter — cheap struct field check
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
		result := make([]*FileIndex, 0, len(indexes))
		for _, idx := range indexes {
			result = append(result, idx)
		}
		return result
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
	set := make(map[string]struct{}, len(levels))
	for _, l := range levels {
		set[l] = struct{}{}
	}
	return set
}
