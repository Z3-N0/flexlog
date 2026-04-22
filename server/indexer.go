package server

import (
	"bufio"
	"context"
	"os"
	"runtime"
	"sync"

	"github.com/Z3-N0/flexlog"
)

// FileIndex holds the byte offsets for every line in a single log file.
type FileIndex struct {
	Path    string
	Offsets []int64 // one entry per line, value is the byte offset of that line
	Count   int     // total lines indexed, including malformed
}

// ProgressFunc is called during indexing with the file path and lines indexed so far. Pass nil if progress reporting is not needed.
type ProgressFunc func(file string, linesIndexed int)

const progressInterval = 1000

// BuildIndex indexes all files in the provided list concurrently and returns a map keyed by relative file path.
func BuildIndex(ctx context.Context, logger *flexlog.Logger, files []string, progress ProgressFunc) map[string]*FileIndex {
	results := make(map[string]*FileIndex, len(files))
	var mu sync.Mutex

	workers := runtime.NumCPU()
	if workers > len(files) {
		workers = len(files)
	}

	logger.Debug(ctx, "starting parallel indexer", "workers", workers, "files", len(files))

	fileCh := make(chan string, len(files))
	for _, f := range files {
		fileCh <- f
	}
	close(fileCh)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileCh {
				idx := indexFile(ctx, logger, path, progress)
				mu.Lock()
				results[path] = idx
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	return results
}

// indexFile does a single linear scan of the file, recording the byte offset of each line.
func indexFile(ctx context.Context, logger *flexlog.Logger, path string, progress ProgressFunc) *FileIndex {
	idx := &FileIndex{
		Path:    path,
		Offsets: make([]int64, 0, 1024),
	}

	f, err := os.Open(path)
	if err != nil {
		logger.Error(ctx, "failed to open file for indexing", "path", path, "error", err.Error())
		return idx
	}
	defer f.Close()

	var offset int64
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024) // 256KB max line size

	for scanner.Scan() {
		idx.Offsets = append(idx.Offsets, offset)
		offset += int64(len(scanner.Bytes())) + 1 // +1 for the newline
		idx.Count++

		if progress != nil && idx.Count%progressInterval == 0 {
			progress(path, idx.Count)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Warn(ctx, "scanner error during indexing", "path", path, "error", err.Error())
	}

	// final progress update
	if progress != nil && idx.Count%progressInterval != 0 {
		progress(path, idx.Count)
	}

	return idx
}

// ReadLine reads a single line from a file at the given byte offset.
func ReadLine(path string, offset int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, os.SEEK_SET); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)
	if scanner.Scan() {
		line := make([]byte, len(scanner.Bytes()))
		copy(line, scanner.Bytes())
		return line, nil
	}
	return nil, scanner.Err()
}
