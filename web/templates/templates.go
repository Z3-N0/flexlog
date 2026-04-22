package templates

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"strings"
)

var (
	//go:embed *.html icons/*.html
	templateFS embed.FS

	Templates *template.Template
)

var funcMap = template.FuncMap{
	"add":   func(a, b int) int { return a + b },
	"sub":   func(a, b int) int { return a - b },
	"lower": strings.ToLower,
	"levelColor": func(level string) string {
		switch strings.ToUpper(level) {
		case "TRACE":
			return "#6b6b80"
		case "DEBUG":
			return "#4a9eff"
		case "INFO":
			return "#34d399"
		case "WARN":
			return "#fbbf24"
		case "ERROR":
			return "#f87171"
		case "FATAL":
			return "#dc2626"
		default:
			return "#6b6b80"
		}
	},
}

// Initialize parses all HTML templates and sets up the static file server.
func Initialize() {
	Templates = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "*.html", "icons/*.html"))
}

// WriteResponse writes an executed template to the response writer.
func WriteResponse(w http.ResponseWriter, name string, data any) error {
	var buf bytes.Buffer
	if err := Templates.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, err := buf.WriteTo(w)
	return err
}
