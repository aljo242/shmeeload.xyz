package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/rs/zerolog/log"
)

type htmlTemplateInfo struct {
	Host string
}

// ExecuteTemplateHTML renders the HTML template at path and writes the result to
// newPath. Pages use origin-relative URLs (an empty Host), so the same output
// works behind any host, port, or scheme without rebuilding.
func ExecuteTemplateHTML(path, newPath string) error {
	filePath := filepath.Clean(newPath)
	newFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file %v : %w", newPath, err)
	}
	defer func() {
		if err := newFile.Close(); err != nil {
			log.Error().Err(err).Str("filename", filePath).Msg("error closing the file")
		}
	}()

	tpl, err := template.ParseFiles(path)
	if err != nil {
		return fmt.Errorf("error creating template : %w", err)
	}

	if err = tpl.Execute(newFile, htmlTemplateInfo{Host: ""}); err != nil {
		return fmt.Errorf("error executing template : %w", err)
	}

	return nil
}
