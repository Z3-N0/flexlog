package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/Z3-N0/flexlog"
	"github.com/Z3-N0/flexlog/server"
	"github.com/Z3-N0/flexlog/web/templates"
)

// Handler holds the dependencies all HTTP handlers need.
type Handler struct {
	indexes map[string]*server.FileIndex
	scan    server.ScanResult
	logger  *flexlog.Logger
	indexed *atomic.Int64
	ready   *atomic.Bool
}

// New creates a Handler with all required dependencies.
func New(
	indexes map[string]*server.FileIndex,
	scan server.ScanResult,
	logger *flexlog.Logger,
	indexed *atomic.Int64,
	ready *atomic.Bool,
) *Handler {
	return &Handler{
		indexes: indexes,
		scan:    scan,
		logger:  logger,
		indexed: indexed,
		ready:   ready,
	}
}

// HandleIndex serves the full page shell.
func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if err := templates.WriteResponse(w, "index.html", nil); err != nil {
		h.serverError(w, err)
	}
}

// HandleLayout serves the page layout fragment.
func (h *Handler) HandleLayout(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Files []string
	}{
		Files: h.scan.Files,
	}
	if err := templates.WriteResponse(w, "pg-layout.html", data); err != nil {
		h.serverError(w, err)
	}
}

// HandleStatus returns an HTML fragment reflecting indexing state.
// If not ready, the fragment includes an HTMX poll so the browser retries automatically.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Ready   bool
		Indexed int64
	}{
		Ready:   h.ready.Load(),
		Indexed: h.indexed.Load(),
	}
	if err := templates.WriteResponse(w, "fg-status.html", data); err != nil {
		h.serverError(w, err)
	}
}

// HandleQuery runs a query and returns a paginated HTML fragment.
func (h *Handler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		http.Error(w, "indexing in progress", http.StatusServiceUnavailable)
		return
	}

	q := parseQuery(r)
	result := server.Execute(q, h.indexes)

	if err := templates.WriteResponse(w, "fg-logs.html", result); err != nil {
		h.serverError(w, err)
	}
}

// HandleRaw reads and returns a single raw log line by file path and byte offset.
func (h *Handler) HandleRaw(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	offsetStr := r.URL.Query().Get("offset")

	if _, ok := h.indexes[file]; !ok {
		http.Error(w, "unknown file", http.StatusBadRequest)
		return
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid offset", http.StatusBadRequest)
		return
	}

	line, err := server.ReadLine(file, offset)
	if err != nil {
		h.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(line)
}

// parseQuery builds a Query from request URL params.
func parseQuery(r *http.Request) server.Query {
	q := server.Query{
		Search:        r.URL.Query().Get("search"),
		TraceID:       r.URL.Query().Get("trace_id"),
		Levels:        r.URL.Query()["level"],
		Files:         r.URL.Query()["file"],
		ShowMalformed: r.URL.Query().Get("malformed") == "true",
		SortDesc:      r.URL.Query().Get("desc") != "false",
		PageSize:      50,
	}

	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			q.Page = n
		}
	}

	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := server.ParseTimeParam(from); err == nil {
			q.From = t
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := server.ParseTimeParam(to); err == nil {
			q.To = t
		}
	}

	return q
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.logger.Error(context.Background(), "handler error", "err", err)
	http.Error(w, fmt.Sprintf("internal error: %v", err), http.StatusInternalServerError)
}
