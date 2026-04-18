// web/static.go
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticEmbed embed.FS

var StaticFS http.FileSystem

func init() {
    sub, _ := fs.Sub(staticEmbed, "static")
    StaticFS = http.FS(sub)
}
