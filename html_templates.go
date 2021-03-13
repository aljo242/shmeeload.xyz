package main

import (
	"fmt"
	"os"
	"text/template"
)

type htmlTemplateInfo struct {
	Host string
	// TODO add more
}

// ExecuteTemplateHTML is a util func for executing an html template
// at path and saving the new file to newPath
func ExecuteTemplateHTML(cfg ServerConfig, path, newPath string) error {
	newFile, err := os.Create(newPath)
	defer newFile.Close()
	if err != nil {
		return fmt.Errorf("error creating file %v : %w", newPath, err)
	}

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
