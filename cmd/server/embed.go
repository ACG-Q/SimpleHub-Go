package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

//go:embed dist/*
var embeddedFS embed.FS

const customDistPath = "./web/dist"

func getDistFS() http.FileSystem {
	if _, err := os.Stat(customDistPath + "/index.html"); err == nil {
		log.Info().Str("path", customDistPath).Msg("using custom frontend from disk")
		return http.Dir(customDistPath)
	}

	log.Info().Msg("using embedded frontend")
	f, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to access embedded frontend")
	}
	return http.FS(f)
}
