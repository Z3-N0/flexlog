package web

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/Z3-N0/flexlog"
	"github.com/Z3-N0/flexlog/server"
	"github.com/Z3-N0/flexlog/web/templates"
)

// App holds all runtime state for the viewer.
type App struct {
	indexes map[string]*server.FileIndex
	scan    server.ScanResult
	logger  *flexlog.Logger
	indexed atomic.Int64
	ready   atomic.Bool
}

// Initialize sets up the App, parses templates, starts background indexing and returns an http.Handler.
func Initialize(ctx context.Context, scan server.ScanResult, logger *flexlog.Logger) (http.Handler, error) {
	templates.Initialize()

	app := &App{
		scan:   scan,
		logger: logger,
	}

	go app.index(ctx)

	return app.Routes(), nil
}

// index runs BuildIndex in the background and marks the app ready when done.
func (a *App) index(ctx context.Context) {
	progress := func(file string, linesIndexed int) {
		a.indexed.Store(int64(linesIndexed))
		a.logger.Info(ctx, "indexing", "file", file, "lines", linesIndexed)
	}

	a.indexes = server.BuildIndex(a.scan.Files, progress)
	a.ready.Store(true)
	a.logger.Info(ctx, "indexing complete", "files", len(a.scan.Files))
}
