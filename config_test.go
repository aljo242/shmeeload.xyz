package main

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid config unmarshals", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "c.json")
		mustWrite(t, p, `{"port":"8080","IP":"0.0.0.0","secure":true,"cacheMaxAge":42}`)
		cfg, err := LoadConfig(p)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.Port != "8080" || cfg.IP != "0.0.0.0" || !cfg.HTTPS || cfg.CacheMaxAge != 42 {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})

	t.Run("missing file preserves fs.ErrNotExist", func(t *testing.T) {
		_, err := LoadConfig(filepath.Join(t.TempDir(), "nope.json"))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("err = %v, want it to wrap fs.ErrNotExist", err)
		}
	})

	t.Run("malformed JSON errors", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "bad.json")
		mustWrite(t, p, "{not valid json")
		if _, err := LoadConfig(p); err == nil {
			t.Fatal("expected an error for malformed JSON")
		}
	})
}
