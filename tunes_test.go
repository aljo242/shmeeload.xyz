package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTuneName(t *testing.T) {
	cases := map[string]string{
		"my_song.mp3":         "my song",
		"track-one-final.MP3": "track one final",
		"plain.mp3":           "plain",
	}
	for in, want := range cases {
		if got := tuneName(in); got != want {
			t.Errorf("tuneName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestListTunes(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"b_song.mp3", "a_song.mp3", "notes.txt", "cover.png"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tunes := listTunes(dir)
	if len(tunes) != 2 {
		t.Fatalf("listTunes returned %d tracks, want 2 (mp3 only)", len(tunes))
	}
	if tunes[0].File != "a_song.mp3" || tunes[1].File != "b_song.mp3" {
		t.Errorf("not sorted by filename: %v", tunes)
	}
	if tunes[0].Name != "a song" || tunes[0].Size != 4 {
		t.Errorf("unexpected tune: %+v", tunes[0])
	}

	if got := listTunes(filepath.Join(dir, "missing")); got != nil {
		t.Errorf("missing dir should give nil, got %v", got)
	}
}

func TestTunesFileHandler(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "song.mp3"), []byte("ID3 fake mp3 bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := tunesFileHandler(dir)

	t.Run("serves an existing mp3 with a range-capable response", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tunes/file/song.mp3", nil)
		req.SetPathValue("name", "song.mp3")
		h(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if !strings.HasPrefix(rr.Header().Get("Content-Type"), "audio/") {
			t.Errorf("content-type = %q, want audio/*", rr.Header().Get("Content-Type"))
		}
		if rr.Header().Get("Accept-Ranges") != "bytes" {
			t.Errorf("missing Accept-Ranges: bytes (needed for streaming/seeking)")
		}
	})

	t.Run("rejects non-mp3 and traversal", func(t *testing.T) {
		for _, name := range []string{"song.txt", "..", "../secret"} {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/tunes/file/x", nil)
			req.SetPathValue("name", name)
			h(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Errorf("name %q: status = %d, want 404", name, rr.Code)
			}
		}
	})
}

func TestTunesListHandler(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "x.mp3"), []byte("d"), 0o644)
	rr := httptest.NewRecorder()
	tunesListHandler(dir)(rr, httptest.NewRequest(http.MethodGet, "/tunes/list", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var got []tune
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(got) != 1 || got[0].File != "x.mp3" {
		t.Errorf("list = %v", got)
	}
}
