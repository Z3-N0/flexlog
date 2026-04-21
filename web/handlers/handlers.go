package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Z3-N0/flexlog"
	"github.com/Z3-N0/flexlog/server"
	"github.com/Z3-N0/flexlog/web/templates"
)

// ViewerApp is the interface handlers use to access app state.
// Defined here to avoid circular imports.
type ViewerApp interface {
	IsReady() bool
	IndexedCount() int64
	GetIndexes() map[string]*server.FileIndex
	GetScan() server.ScanResult
	GetLogger() *flexlog.Logger
	GetPageSize() int
}

// Handler holds a reference to the live app state.
type Handler struct {
	app ViewerApp
}

// New creates a Handler with the given app.
func New(app ViewerApp) *Handler {
	return &Handler{app: app}
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
		Leftbar struct{ Files []string }
	}{
		Leftbar: struct{ Files []string }{Files: h.app.GetScan().Files},
	}
	if err := templates.WriteResponse(w, "pg-layout.html", data); err != nil {
		h.serverError(w, err)
	}
}

// HandleStatus returns fg-filters.html when ready, fg-status.html with polling while indexing.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if h.app.IsReady() {
		if err := templates.WriteResponse(w, "fg-filters.html", nil); err != nil {
			h.serverError(w, err)
		}
		return
	}
	data := struct {
		Indexed int64
	}{
		Indexed: h.app.IndexedCount(),
	}
	if err := templates.WriteResponse(w, "fg-status.html", data); err != nil {
		h.serverError(w, err)
	}
}

// HandleQuery runs a query and returns a paginated HTML fragment.
func (h *Handler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if !h.app.IsReady() {
		http.Error(w, "indexing in progress", http.StatusServiceUnavailable)
		return
	}

	q, err := parseQuery(r, h.app.GetPageSize())
	if err != nil {
		http.Error(w, "Error Parsing query", http.StatusBadRequest)
		return
	}
	result := server.Execute(q, h.app.GetIndexes())

	if err := templates.WriteResponse(w, "fg-logs.html", result); err != nil {
		h.serverError(w, err)
	}
}

// HandleRaw reads and returns a single raw log line by file path and byte offset.
func (h *Handler) HandleRaw(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	offsetStr := r.URL.Query().Get("offset")

	if _, ok := h.app.GetIndexes()[file]; !ok {
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
	_, err = w.Write(line)
	if err != nil {
		h.serverError(w, err)
		return
	}
}

// parseQuery builds a Query from request URL params.
func parseQuery(r *http.Request, pageSize int) (server.Query, error) {
	q := server.Query{}
	err := r.ParseForm()
	if err != nil {
		return q, err
	}

	q = server.Query{
		Search:        r.FormValue("search"),
		TraceID:       r.FormValue("trace_id"),
		Levels:        r.Form["level"],
		Files:         r.Form["file"],
		ShowMalformed: r.FormValue("malformed") == "true",
		SortDesc:      r.FormValue("desc") != "false",
		PageSize:      pageSize,
	}

	if p := r.FormValue("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			q.Page = n
		}
	}

	if from := r.FormValue("from"); from != "" {
		if t, err := server.ParseTimeParam(from); err == nil {
			q.From = t
		}
	}
	if to := r.FormValue("to"); to != "" {
		if t, err := server.ParseTimeParam(to); err == nil {
			q.To = t
		}
	}

	return q, nil
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.app.GetLogger().Error(context.Background(), "handler error", "err", err)
	http.Error(w, fmt.Sprintf("internal error: %v", err), http.StatusInternalServerError)
}
