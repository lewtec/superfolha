//go:build release

package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web/dist
var webDist embed.FS

func GetWebApp() http.Handler {
	dist, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(dist))
}
