package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// defaultTunesDir is where MP3s are read from when the config leaves it empty.
// It is a read-only bind mount of a host folder, so songs can be added by
// dropping files in without rebuilding the binary.
const defaultTunesDir = "/tunes"

func tunesDirOf(cfg Config) string {
	if cfg.TunesDir != "" {
		return cfg.TunesDir
	}
	return defaultTunesDir
}

// tune is one track in the listing.
type tune struct {
	File string `json:"file"` // filename, used in the /tunes/file/<file> URL
	Name string `json:"name"` // display name derived from the filename
	Size int64  `json:"size"` // bytes
}

// tuneName turns a filename into a display name: drop the extension and turn
// underscores and dashes into spaces.
func tuneName(file string) string {
	name := strings.TrimSuffix(file, filepath.Ext(file))
	name = strings.NewReplacer("_", " ", "-", " ").Replace(name)
	return strings.TrimSpace(name)
}

// listTunes returns the .mp3 files in dir, sorted by filename. A missing
// directory yields an empty list, not an error.
func listTunes(dir string) []tune {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var tunes []tune
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".mp3") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		tunes = append(tunes, tune{File: e.Name(), Name: tuneName(e.Name()), Size: info.Size()})
	}
	sort.Slice(tunes, func(i, j int) bool { return tunes[i].File < tunes[j].File })
	return tunes
}

// tunesListHandler serves the track listing as JSON.
func tunesListHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		tunes := listTunes(dir)
		if tunes == nil {
			tunes = []tune{}
		}
		_ = json.NewEncoder(w).Encode(tunes)
	}
}

// tunesFileHandler streams a single MP3 with Range support (so browsers can seek
// and stream). The name must be a single .mp3 filename within the tunes dir.
func tunesFileHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" || name != filepath.Base(name) || !strings.HasSuffix(strings.ToLower(name), ".mp3") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeFile(w, r, filepath.Join(dir, name)) // sets audio/mpeg + handles Range
	}
}
