package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAssets(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "home.css"), "body{}")
	mustWrite(t, filepath.Join(base, "home.html"), `<base href="{{.Host}}/"/><link href="{{.Host}}/static/css/home.css"/>`)
	mustWrite(t, filepath.Join(base, "img", "icon", "1", "favicon.ico"), "ico") // nested
	mustWrite(t, filepath.Join(base, "dist", "app.js"), "console.log(1)")       // tsc output
	mustWrite(t, filepath.Join(base, "files", "resume.pdf"), "%PDF")
	mustWrite(t, filepath.Join(base, "manifest.json"), `{"x":1}`)
	mustWrite(t, filepath.Join(base, "node_modules", "junk.js"), "junk") // must be skipped

	assets, err := buildAssets(base)
	if err != nil {
		t.Fatalf("buildAssets: %v", err)
	}

	// Exact-key assets, including the nested favicon flattened into img/.
	for key, want := range map[string]string{
		"css/home.css":     "body{}",
		"img/favicon.ico":  "ico",
		"js/app.js":        "console.log(1)",
		"files/resume.pdf": "%PDF",
		"manifest.json":    `{"x":1}`,
	} {
		got, ok := assets[key]
		if !ok {
			t.Errorf("missing asset %q", key)
			continue
		}
		if string(got) != want {
			t.Errorf("asset %q = %q, want %q", key, got, want)
		}
	}

	// HTML is rendered with an empty Host, so URLs are origin-relative.
	html, ok := assets["html/home.html"]
	if !ok {
		t.Fatal("missing html/home.html")
	}
	if !strings.Contains(string(html), `<base href="/"/>`) {
		t.Errorf("home.html not rendered with empty host: %q", html)
	}

	// node_modules must be skipped entirely.
	if _, ok := assets["js/junk.js"]; ok {
		t.Error("node_modules content should be skipped")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
