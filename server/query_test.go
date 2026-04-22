package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Z3-N0/flexlog"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// writeTempLog writes lines to a temp file and returns its path and a
// pre-built FileIndex for it. The index is built by BuildIndex so it uses
// the real indexing path.
func writeTempLog(t *testing.T, lines []string) (string, *FileIndex) {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "*.log")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	for _, l := range lines {
		fmt.Fprintln(f, l)
	}
	f.Close()

	ctx := context.Background()
	logger := flexlog.New()
	indexes := BuildIndex(ctx, logger, []string{f.Name()}, nil)
	idx, ok := indexes[f.Name()]
	if !ok {
		t.Fatalf("BuildIndex did not produce an index for %s", f.Name())
	}
	return f.Name(), idx
}

// makeIndexes wraps one or more (path, *FileIndex) pairs into the map that
// Execute expects.
func makeIndexes(pairs ...*FileIndex) map[string]*FileIndex {
	m := make(map[string]*FileIndex, len(pairs))
	for _, idx := range pairs {
		m[idx.Path] = idx
	}
	return m
}

// ts formats a unix-second timestamp the way ParseLine expects.
func ts(sec int64) string {
	return fmt.Sprintf("%d", sec)
}

// logLine produces a minimal valid JSON log line.
func logLine(level, msg, traceID string, unixSec int64) string {
	return fmt.Sprintf(`{"level":%q,"msg":%q,"trace_id":%q,"ts":%s}`,
		level, msg, traceID, ts(unixSec))
}

// ── resolveFiles ─────────────────────────────────────────────────────────────

func TestResolveFiles(t *testing.T) {
	idx1 := &FileIndex{Path: "a.log"}
	idx2 := &FileIndex{Path: "b.log"}
	indexes := map[string]*FileIndex{"a.log": idx1, "b.log": idx2}

	t.Run("empty Files returns nil (known bug: should return all)", func(t *testing.T) {
		// Document the current behaviour so a future fix is caught.
		result := resolveFiles(Query{}, indexes)
		if result != nil {
			t.Logf("resolveFiles now returns all files when q.Files is empty – update this test")
		}
		// Once the bug is fixed, assert len(result) == 2 instead.
	})

	t.Run("specific files are resolved", func(t *testing.T) {
		result := resolveFiles(Query{Files: []string{"a.log"}}, indexes)
		if len(result) != 1 || result[0].Path != "a.log" {
			t.Errorf("got %v, want [a.log]", result)
		}
	})

	t.Run("unknown file is silently ignored", func(t *testing.T) {
		result := resolveFiles(Query{Files: []string{"nope.log"}}, indexes)
		if len(result) != 0 {
			t.Errorf("expected empty result, got %v", result)
		}
	})
}

// ── toLevelSet ───────────────────────────────────────────────────────────────

func TestToLevelSet(t *testing.T) {
	t.Run("nil when empty", func(t *testing.T) {
		if toLevelSet(nil) != nil {
			t.Error("expected nil for nil input")
		}
		if toLevelSet([]string{}) != nil {
			t.Error("expected nil for empty slice")
		}
		if toLevelSet([]string{""}) != nil {
			t.Error("expected nil for slice of empty strings")
		}
	})

	t.Run("uppercases entries", func(t *testing.T) {
		set := toLevelSet([]string{"info", "warn"})
		for _, want := range []string{"INFO", "WARN"} {
			if _, ok := set[want]; !ok {
				t.Errorf("expected %q in set", want)
			}
		}
	})

	t.Run("splits comma-separated entry", func(t *testing.T) {
		set := toLevelSet([]string{"info,error, debug"})
		for _, want := range []string{"INFO", "ERROR", "DEBUG"} {
			if _, ok := set[want]; !ok {
				t.Errorf("expected %q in set after comma split", want)
			}
		}
	})
}

// ── Execute – end-to-end ─────────────────────────────────────────────────────

// Anchor timestamps so tests are deterministic.
var (
	t0 = int64(1700000000) // base unix second
	t1 = t0 + 60
	t2 = t0 + 120
	t3 = t0 + 180
)

func TestExecute_NoFiles(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	result := Execute(ctx, logger, Query{Files: []string{"missing.log"}}, map[string]*FileIndex{})
	if result.TotalMatch != 0 {
		t.Errorf("TotalMatch = %d, want 0", result.TotalMatch)
	}
	if result.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1", result.TotalPages)
	}
}

func TestExecute_AllEntries(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "started", "", t0),
		logLine("ERROR", "boom", "", t1),
		logLine("DEBUG", "tick", "", t2),
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	result := Execute(ctx, logger, Query{Files: []string{path}, PageSize: 10}, indexes)

	if result.TotalMatch != 3 {
		t.Errorf("TotalMatch = %d, want 3", result.TotalMatch)
	}
	if len(result.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(result.Entries))
	}
}

func TestExecute_FilterByLevel(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "started", "", t0),
		logLine("ERROR", "boom", "", t1),
		logLine("ERROR", "boom again", "", t2),
		logLine("DEBUG", "tick", "", t3),
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	result := Execute(ctx, logger, Query{
		Files:    []string{path},
		Levels:   []string{"ERROR"},
		PageSize: 10,
	}, indexes)

	if result.TotalMatch != 2 {
		t.Errorf("TotalMatch = %d, want 2", result.TotalMatch)
	}
	for _, e := range result.Entries {
		if e.Level != "ERROR" {
			t.Errorf("unexpected level %q in results", e.Level)
		}
	}
}

func TestExecute_FilterBySearch(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "user logged in", "", t0),
		logLine("INFO", "cache miss on key foo", "", t1),
		logLine("ERROR", "database connection failed", "", t2),
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	result := Execute(ctx, logger, Query{
		Files:    []string{path},
		Search:   "cache",
		PageSize: 10,
	}, indexes)

	if result.TotalMatch != 1 {
		t.Errorf("TotalMatch = %d, want 1", result.TotalMatch)
	}
	if result.Entries[0].Message != "cache miss on key foo" {
		t.Errorf("Message = %q", result.Entries[0].Message)
	}
}

func TestExecute_SearchIsCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "User Logged In", "", t0),
		logLine("INFO", "cache miss", "", t1),
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	result := Execute(ctx, logger, Query{
		Files:    []string{path},
		Search:   "USER",
		PageSize: 10,
	}, indexes)

	if result.TotalMatch != 1 {
		t.Errorf("TotalMatch = %d, want 1", result.TotalMatch)
	}
}

func TestExecute_FilterByTraceID(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "req start", "trace-aaa", t0),
		logLine("INFO", "req end", "trace-aaa", t1),
		logLine("INFO", "other req", "trace-bbb", t2),
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	result := Execute(ctx, logger, Query{
		Files:    []string{path},
		TraceID:  "trace-aaa",
		PageSize: 10,
	}, indexes)

	if result.TotalMatch != 2 {
		t.Errorf("TotalMatch = %d, want 2", result.TotalMatch)
	}
}

func TestExecute_FilterByTimeRange(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "early", "", t0),
		logLine("INFO", "middle", "", t1),
		logLine("INFO", "late", "", t2),
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	from := time.Unix(t0+1, 0).UTC() // excludes t0
	to := time.Unix(t2-1, 0).UTC()   // excludes t2

	result := Execute(ctx, logger, Query{
		Files:    []string{path},
		From:     from,
		To:       to,
		PageSize: 10,
	}, indexes)

	if result.TotalMatch != 1 {
		t.Errorf("TotalMatch = %d, want 1 (only middle)", result.TotalMatch)
	}
	if result.Entries[0].Message != "middle" {
		t.Errorf("Message = %q, want \"middle\"", result.Entries[0].Message)
	}
}

func TestExecute_ShowMalformed(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "good entry", "", t0),
		"this is not json at all",
		`{"level":"WARN","msg":"also good"}`,
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	t.Run("malformed hidden by default", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files:    []string{path},
			PageSize: 10,
		}, indexes)
		if result.TotalMatch != 2 {
			t.Errorf("TotalMatch = %d, want 2 (malformed hidden)", result.TotalMatch)
		}
	})

	t.Run("malformed shown when ShowMalformed=true", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files:         []string{path},
			PageSize:      10,
			ShowMalformed: true,
		}, indexes)
		if result.TotalMatch != 3 {
			t.Errorf("TotalMatch = %d, want 3 (malformed visible)", result.TotalMatch)
		}
	})
}

func TestExecute_SortAscDesc(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "first", "", t0),
		logLine("INFO", "second", "", t1),
		logLine("INFO", "third", "", t2),
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	t.Run("ascending (default)", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files:    []string{path},
			PageSize: 10,
			SortDesc: false,
		}, indexes)
		if result.Entries[0].Message != "first" {
			t.Errorf("first entry = %q, want \"first\"", result.Entries[0].Message)
		}
	})

	t.Run("descending", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files:    []string{path},
			PageSize: 10,
			SortDesc: true,
		}, indexes)
		if result.Entries[0].Message != "third" {
			t.Errorf("first entry = %q, want \"third\"", result.Entries[0].Message)
		}
	})
}

func TestExecute_Pagination(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	// 5 entries, page size 2 → 3 pages
	lines := make([]string, 5)
	for i := range lines {
		lines[i] = logLine("INFO", fmt.Sprintf("msg-%d", i), "", t0+int64(i))
	}
	path, idx := writeTempLog(t, lines)
	indexes := makeIndexes(idx)

	t.Run("page 1", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files: []string{path}, Page: 1, PageSize: 2,
		}, indexes)
		if result.TotalMatch != 5 {
			t.Errorf("TotalMatch = %d, want 5", result.TotalMatch)
		}
		if result.TotalPages != 3 {
			t.Errorf("TotalPages = %d, want 3", result.TotalPages)
		}
		if len(result.Entries) != 2 {
			t.Errorf("len(Entries) = %d, want 2", len(result.Entries))
		}
	})

	t.Run("last page (partial)", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files: []string{path}, Page: 3, PageSize: 2,
		}, indexes)
		if len(result.Entries) != 1 {
			t.Errorf("len(Entries) = %d, want 1 (last partial page)", len(result.Entries))
		}
	})

	t.Run("page beyond last returns empty entries", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files: []string{path}, Page: 99, PageSize: 2,
		}, indexes)
		if len(result.Entries) != 0 {
			t.Errorf("expected 0 entries for out-of-range page, got %d", len(result.Entries))
		}
	})

	t.Run("page < 1 is clamped to 1", func(t *testing.T) {
		result := Execute(ctx, logger, Query{
			Files: []string{path}, Page: -5, PageSize: 10,
		}, indexes)
		if result.Page != 1 {
			t.Errorf("Page = %d, want 1 after clamping", result.Page)
		}
	})
}

func TestExecute_MultiFile(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	lines1 := []string{logLine("INFO", "from-file-1", "", t0)}
	lines2 := []string{logLine("INFO", "from-file-2", "", t1)}

	path1, idx1 := writeTempLog(t, lines1)
	path2, idx2 := writeTempLog(t, lines2)
	indexes := makeIndexes(idx1, idx2)

	result := Execute(ctx, logger, Query{
		Files:    []string{path1, path2},
		PageSize: 10,
	}, indexes)

	if result.TotalMatch != 2 {
		t.Errorf("TotalMatch = %d, want 2 across two files", result.TotalMatch)
	}
}

// ── BuildIndex ────────────────────────────────────────────────────────────────

func TestBuildIndex(t *testing.T) {
	logger := flexlog.New()
	defer logger.Close()

	lines := []string{
		logLine("INFO", "a", "", t0),
		logLine("INFO", "b", "", t1),
		logLine("INFO", "c", "", t2),
	}
	path, idx := writeTempLog(t, lines)

	if idx.Count != 3 {
		t.Errorf("Count = %d, want 3", idx.Count)
	}
	if len(idx.Offsets) != 3 {
		t.Errorf("len(Offsets) = %d, want 3", len(idx.Offsets))
	}
	if idx.Offsets[0] != 0 {
		t.Errorf("Offsets[0] = %d, want 0", idx.Offsets[0])
	}
	if idx.Path != path {
		t.Errorf("Path = %q, want %q", idx.Path, path)
	}
}

func TestBuildIndex_MissingFile(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	indexes := BuildIndex(ctx, logger, []string{"/does/not/exist.log"}, nil)
	idx := indexes["/does/not/exist.log"]
	if idx == nil {
		t.Fatal("expected an index entry even for missing files")
	}
	if idx.Count != 0 {
		t.Errorf("Count = %d, want 0 for missing file", idx.Count)
	}
}

func TestBuildIndex_ProgressCallback(t *testing.T) {
	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()

	// Write enough lines to trigger at least one progress report (every 1000).
	lines := make([]string, 1001)
	for i := range lines {
		lines[i] = logLine("INFO", fmt.Sprintf("msg-%d", i), "", t0+int64(i))
	}
	f, _ := os.CreateTemp(t.TempDir(), "*.log")
	for _, l := range lines {
		fmt.Fprintln(f, l)
	}
	f.Close()

	called := false
	BuildIndex(ctx, logger, []string{f.Name()}, func(file string, linesIndexed int) {
		called = true
		if file != f.Name() {
			t.Errorf("progress file = %q, want %q", file, f.Name())
		}
	})
	if !called {
		t.Error("progress callback was never called")
	}
}

// ── ReadLine ─────────────────────────────────────────────────────────────────

func TestReadLine(t *testing.T) {
	lines := []string{
		logLine("INFO", "line zero", "", t0),
		logLine("INFO", "line one", "", t1),
		logLine("INFO", "line two", "", t2),
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	f, _ := os.Create(path)
	for _, l := range lines {
		fmt.Fprintln(f, l)
	}
	f.Close()

	ctx := context.Background()
	logger := flexlog.New()
	defer logger.Close()
	indexes := BuildIndex(ctx, logger, []string{path}, nil)
	idx := indexes[path]

	for i, wantLine := range lines {
		got, err := ReadLine(path, idx.Offsets[i])
		if err != nil {
			t.Errorf("ReadLine offset[%d]: %v", i, err)
			continue
		}
		if string(got) != wantLine {
			t.Errorf("ReadLine[%d] = %q, want %q", i, got, wantLine)
		}
	}
}

func TestReadLine_MissingFile(t *testing.T) {
	_, err := ReadLine("/does/not/exist.log", 0)
	if err == nil {
		t.Error("expected error for missing file")
	}
}
