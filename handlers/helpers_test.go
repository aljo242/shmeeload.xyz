package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

// setupStatic points the handler package at a fresh temp static tree (with all
// the per-type subdirectories created) and restores the defaults when the test
// ends. It returns the root so tests can drop files into it.
func setupStatic(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, sub := range []string{"html", "js", "css", "src", "img", "model", "files"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	SetStaticRoot(root)
	SetSiteRoot(root)
	t.Cleanup(func() {
		SetStaticRoot("./static")
		SetSiteRoot(".")
	})
	return root
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
