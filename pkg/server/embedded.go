package server

import (
	"embed"
	"io/fs"
)

//go:embed ui/dist/*
var embeddedWebFS embed.FS

// GetEmbeddedWebFS returns an fs.FS sub-rooted at ui/dist for standalone single-binary serving.
func GetEmbeddedWebFS() fs.FS {
	sub, err := fs.Sub(embeddedWebFS, "ui/dist")
	if err != nil {
		return nil
	}
	return sub
}
