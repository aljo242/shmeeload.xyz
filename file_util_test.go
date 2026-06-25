package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExists(t *testing.T) {
	dir := t.TempDir()
	if !Exists(dir) {
		t.Errorf("Exists(%q) = false, want true", dir)
	}
	missing := filepath.Join(dir, "does-not-exist")
	if Exists(missing) {
		t.Errorf("Exists(%q) = true, want false", missing)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "created")

	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if !Exists(dir) {
		t.Fatalf("EnsureDir did not create %q", dir)
	}

	// EnsureDir must be idempotent.
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir (second call): %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeTestFile(t, src, "hello world")

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("copied content = %q, want %q", got, "hello world")
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "nope.txt"), filepath.Join(dir, "dst.txt"))
	if err == nil {
		t.Fatal("CopyFile with missing source returned nil error, want error")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
