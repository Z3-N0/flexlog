package web

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/Z3-N0/flexlog"
	"github.com/Z3-N0/flexlog/server"
	"github.com/Z3-N0/flexlog/web/templates"
)

// App holds all runtime state for the viewer.
type App struct {
	indexes  map[string]*server.FileIndex
	scan     server.ScanResult
	logger   *flexlog.Logger
	indexed  atomic.Int64
	ready    atomic.Bool
	pageSize int
	port     int
}

// Initialize sets up the App, parses templates, starts background indexing and returns an http.Handler.
func Initialize(ctx context.Context, scan server.ScanResult, logger *flexlog.Logger, pageSize int, port int) (http.Handler, error) {
	templates.Initialize()

	app := &App{
		scan:     scan,
		logger:   logger,
		pageSize: pageSize,
		port:     port,
	}

	go app.index(ctx)

	return app.Routes(), nil
}

func (a *App) GetPageSize() int { return a.pageSize }

// index runs BuildIndex in the background and marks the app ready when done.
func (a *App) index(ctx context.Context) {
	files := a.scan.Files
	counts := make(map[string]int, len(files))
	var mu sync.Mutex // Add this to prevent interleaved prints

	// Initial print to set up the lines
	for _, f := range files {
		fmt.Printf("  %-40s 0 lines\n", f)
	}

	progress := func(file string, linesIndexed int) {
		mu.Lock()
		defer mu.Unlock()

		a.indexed.Add(int64(linesIndexed - counts[file]))
		counts[file] = linesIndexed

		// Move up to the start of our list
		fmt.Printf("\033[%dA", len(files))
		for _, f := range files {
			// Clear line and reprint
			fmt.Printf("\r\033[K  %-40s %d lines\n", f, counts[f])
		}
	}

	a.logger.Debug(ctx, "starting background indexing", "count", len(files))
	// This blocks until indexing is 100% finished
	a.indexes = server.BuildIndex(ctx, a.logger, a.scan.Files, progress)
	a.ready.Store(true)

	fmt.Println() // One line of padding
	a.logger.Info(ctx, "indexing complete", "files", len(a.scan.Files))

	fmt.Print(`
 ______   __         ______     __  __     __         ______     ______
/\  ___\ /\ \       /\  ___\   /\_\_\_\   /\ \       /\  __ \   /\  ___\
\ \  __\ \ \ \____  \ \  __\   \/_/\_\/_  \ \ \____  \ \ \/\ \  \ \ \__ \
 \ \_\    \ \_____\  \ \_____\   /\_\/\_\  \ \_____\  \ \_____\  \ \_____\
  \/_/     \/_____/   \/_____/   \/_/\/_/   \/_____/   \/_____/   \/_____/

`)
	fmt.Printf("\033[1m\033[38;5;135m  ▶  viewer live at http://localhost:%d\033[0m\n\n", a.port)
}

// IsReady returns true when all files are fully indexed.
func (a *App) IsReady() bool { return a.ready.Load() }

// IndexedCount returns the number of lines indexed so far.
func (a *App) IndexedCount() int64 { return a.indexed.Load() }

// GetIndexes returns the live index map.
func (a *App) GetIndexes() map[string]*server.FileIndex { return a.indexes }

// GetScan returns the scan result.
func (a *App) GetScan() server.ScanResult { return a.scan }

// GetLogger returns the logger.
func (a *App) GetLogger() *flexlog.Logger { return a.logger }
