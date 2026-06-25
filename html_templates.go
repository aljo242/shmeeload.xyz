package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/aljo242/chef"
	"github.com/rs/zerolog/log"
)

type htmlTemplateInfo struct {
	Host string
	// TODO add more
}

// ExecuteTemplateHTML is a util func for executing an html template
// at path and saving the new file to newPath
func ExecuteTemplateHTML(cfg chef.ServerConfig, path, newPath string) error {
	filePath := filepath.Clean(newPath)
	newFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file %v : %w", newPath, err)
	}
	defer func() {
		err := newFile.Close()
		if err != nil {
			log.Error().Err(err).Str("filename", filePath).Msg("error closing the file")
		}
	}()

	tpl, err := template.ParseFiles(path)
	if err != nil {
		return fmt.Errorf("error creating template : %w", err)
	}

	// Use origin-relative URLs so the rendered pages work behind any host,
	// port, or scheme (direct LAN, reverse proxy, tailnet, public domain)
	// without rebuilding. An empty Host makes <base href="{{.Host}}/"> render
	// as <base href="/">, so every asset path resolves against the current origin.
	p := htmlTemplateInfo{Host: ""}

	err = tpl.Execute(newFile, p)
	if err != nil {
		return fmt.Errorf("error executing template : %w", err)
	}

	return nil
}
