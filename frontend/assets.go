package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/*
var embeddedAssets embed.FS

func ReadIndex() ([]byte, error) {
	return embeddedAssets.ReadFile("assets/index.html")
}

func StaticHandler() http.Handler {
	staticFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(staticFS))
}
