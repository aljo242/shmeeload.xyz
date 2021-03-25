package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/aljo242/http_util"
	"github.com/rs/zerolog/log"
)

type htmlTemplateInfo struct {
	Host string
	// TODO add more
}

// ExecuteTemplateHTML is a util func for executing an html template
// at path and saving the new file to newPath
func ExecuteTemplateHTML(cfg http_util.ServerConfig, path, newPath string) error {
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

	var httpPrefix string
	if cfg.HTTPS {
		httpPrefix = "https://"
	} else {
		httpPrefix = "http://"
	}

	p := htmlTemplateInfo{httpPrefix + cfg.Host}

	err = tpl.Execute(newFile, p)
	if err != nil {
		return fmt.Errorf("error executing template : %w", err)
	}

	return nil
}
