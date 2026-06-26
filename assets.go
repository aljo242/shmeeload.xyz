package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"
)

// buildAssets walks the web resources directory once and returns an in-memory
// map of served assets keyed by "<type>/<basename>" (html/, js/, css/, src/,
// img/, model/, files/) plus bare filenames for root-level files (manifest.json,
// serviceWorker.js). HTML files are rendered as templates with an empty Host so
// their URLs are origin-relative.
//
// Serving from memory means the container needs no writable filesystem and we
// avoid the old destructive ./static rebuild on every startup.
func buildAssets(baseDir string) (map[string][]byte, error) {
	assets := make(map[string][]byte)

	err := filepath.WalkDir(baseDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(p)
		var key string
		switch filepath.Ext(p) {
		case ".html":
			rendered, err := renderHTML(p)
			if err != nil {
				return err
			}
			assets[path.Join("html", base)] = rendered
			return nil
		case ".js", ".map":
			// The service worker must live at the site root to control the whole scope.
			if base == "serviceWorker.js" || base == "serviceWorker.js.map" {
				key = base
			} else {
				key = path.Join("js", base)
			}
		case ".css":
			key = path.Join("css", base)
		case ".ts":
			key = path.Join("src", base)
		case ".ico", ".png", ".jpg", ".jpeg", ".svg", ".gif":
			key = path.Join("img", base)
		case ".pdf", ".doc", ".docx", ".xml":
			key = path.Join("files", base)
		case ".dae", ".obj", ".gltf":
			key = path.Join("model", base)
		case ".json":
			if base != "manifest.json" {
				return nil
			}
			key = base
		default:
			return nil
		}

		content, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		assets[key] = content
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("building assets from %s: %w", baseDir, err)
	}
	return assets, nil
}

// renderHTML parses and executes the HTML template at p with an empty Host,
// yielding origin-relative URLs.
func renderHTML(p string) ([]byte, error) {
	tpl, err := template.ParseFiles(p)
	if err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", p, err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, struct{ Host string }{Host: ""}); err != nil {
		return nil, fmt.Errorf("rendering template %s: %w", p, err)
	}
	return buf.Bytes(), nil
}
