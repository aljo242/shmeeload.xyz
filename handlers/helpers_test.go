package handlers

import (
	"testing"
	"time"
)

// setupAssets installs an in-memory asset map for the duration of the test and
// restores an empty map afterward. Keys mirror the served layout: "<dir>/<name>"
// (e.g. "css/home.css", "html/home.html") and bare names for site-root files
// (e.g. "manifest.json").
func setupAssets(t *testing.T, files map[string]string) {
	t.Helper()
	m := make(map[string][]byte, len(files))
	for k, v := range files {
		m[k] = []byte(v)
	}
	SetAssets(m, time.Time{})
	t.Cleanup(func() { SetAssets(map[string][]byte{}, time.Time{}) })
}
