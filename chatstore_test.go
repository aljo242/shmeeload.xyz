package main

import (
	"testing"
	"time"
)

func TestChatStore(t *testing.T) {
	s, err := newChatStore(t.TempDir() + "/chat.db")
	if err != nil {
		t.Fatalf("newChatStore: %v", err)
	}
	defer s.close()

	for _, m := range []string{"a: hi", "b: yo"} {
		if err := s.save("general", []byte(m)); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	if err := s.save("music", []byte("c: track")); err != nil {
		t.Fatalf("save: %v", err)
	}

	t.Run("recent returns a room's messages oldest-first", func(t *testing.T) {
		got, err := s.recent("general", 10)
		if err != nil {
			t.Fatalf("recent: %v", err)
		}
		if len(got) != 2 || string(got[0]) != "a: hi" || string(got[1]) != "b: yo" {
			t.Fatalf("recent(general) = %q", got)
		}
	})

	t.Run("recent is scoped per room", func(t *testing.T) {
		if got, _ := s.recent("music", 10); len(got) != 1 {
			t.Fatalf("recent(music) len = %d, want 1", len(got))
		}
		if got, _ := s.recent("art", 10); len(got) != 0 {
			t.Fatalf("recent(art) len = %d, want 0", len(got))
		}
	})

	t.Run("recent honors the limit (keeping the newest)", func(t *testing.T) {
		got, _ := s.recent("general", 1)
		if len(got) != 1 || string(got[0]) != "b: yo" {
			t.Fatalf("recent(general,1) = %q, want newest only", got)
		}
	})

	t.Run("purge keeps recent, deletes past the window", func(t *testing.T) {
		if n, err := s.purgeOlderThan(24 * time.Hour); err != nil || n != 0 {
			t.Fatalf("purge(24h) = %d (err %v), want 0", n, err)
		}
		// A negative window puts the cutoff in the future, so everything is old.
		if n, err := s.purgeOlderThan(-time.Hour); err != nil || n != 3 {
			t.Fatalf("purge(-1h) = %d (err %v), want 3", n, err)
		}
		if got, _ := s.recent("general", 10); len(got) != 0 {
			t.Fatalf("after purge, recent(general) = %q, want empty", got)
		}
	})
}
