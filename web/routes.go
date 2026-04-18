package web

import (
	"net/http"

	"github.com/Z3-N0/flexlog/web/handlers"
)

// Routes registers all HTTP routes and returns the handler.
func (a *App) Routes() http.Handler {
	h := handlers.New(a.indexes, a.scan, a.logger, &a.indexed, &a.ready)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", h.IndexHandler)
	mux.HandleFunc("GET /layout", h.LayoutHandler)
	// mux.HandleFunc("GET /query", h.HandleQuery)
	// mux.HandleFunc("GET /status", h.HandleStatus)
	// mux.HandleFunc("GET /raw", h.HandleRaw)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(StaticFS)))

	return mux
}
