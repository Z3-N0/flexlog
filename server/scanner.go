package server

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Z3-N0/flexlog"
)

// ScanResult holds discovered log files sorted by first-entry timestamp. Relative paths from --dir, sorted chronologically
type ScanResult struct {
	Files []string
}

// ScanDir scans a directory for .log files and sorts them by first-entry timestamp.
func ScanDir(ctx context.Context, logger *flexlog.Logger, dir string) (ScanResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ScanResult{}, err
	}

	type fileWithTime struct {
		path      string
		firstTime time.Time
	}

	var found []fileWithTime

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".log" {
			continue
		}

		rel := filepath.Join(dir, e.Name())
		t := firstEntryTime(ctx, logger, rel)
		if t.IsZero() {
			// fall back to file mtime
			if info, err := e.Info(); err == nil {
				t = info.ModTime()
			}
		}
		found = append(found, fileWithTime{path: rel, firstTime: t})
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].firstTime.IsZero() {
			return false
		}
		if found[j].firstTime.IsZero() {
			return true
		}
		return found[i].firstTime.Before(found[j].firstTime)
	})

	result := ScanResult{Files: make([]string, len(found))}
	for i, f := range found {
		result.Files[i] = f.path
	}
	return result, nil
}

// ScanFile wraps a single file path into a ScanResult for --file mode.
func ScanFile(path string) (ScanResult, error) {
	if _, err := os.Stat(path); err != nil {
		return ScanResult{}, err
	}
	return ScanResult{Files: []string{path}}, nil
}

// firstEntryTime reads the first parseable line of a file and returns its timestamp, zero time if no parseable line is found.
func firstEntryTime(ctx context.Context, logger *flexlog.Logger, path string) time.Time {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	for scanner.Scan() {
		entry := ParseLine(ctx, logger, scanner.Bytes(), path, 0)
		if !entry.Malformed && !entry.Timestamp.IsZero() {
			return entry.Timestamp
		}
	}
	return time.Time{}
}
