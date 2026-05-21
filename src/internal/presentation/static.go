package presentation

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed frontend/*
var frontendFS embed.FS

func frontendFileServer() http.Handler {
	sub, _ := fs.Sub(frontendFS, "frontend")
	return http.FileServer(http.FS(sub))
}
