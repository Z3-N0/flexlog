package templates

import (
	"embed"
	"html/template"
	"net/http"
)

var (
	//go:embed *.html
	templateFS embed.FS

	Templates *template.Template
)

// Initialize parses all HTML templates and sets up the static file server.
func Initialize() {
	Templates = template.Must(template.New("").ParseFS(templateFS, "*.html"))
}

// WriteResponse writes an executed template to the response writer.
func WriteResponse(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	return Templates.ExecuteTemplate(w, name, data)
}
