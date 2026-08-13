package main

import (
	"testing"
	"time"
)

func TestValheimValidBounds(t *testing.T) {
	t.Parallel()
	ok := valheimStatus{Up: true, World: "shmeeland", Seed: "U99eHjhrJ7", Players: []string{"Bjorn"}}
	if !ok.valid() {
		t.Fatal("a normal payload should validate")
	}

	// Negative counters indicate a broken sender, not a quiet server.
	for name, bad := range map[string]valheimStatus{
		"negative uptime":  {UptimeSec: -1},
		"negative players": {PlayerCount: -1},
		"negative backups": {BackupCount: -1},
		"negative age":     {BackupAgeH: -1},
		"negative size":    {WorldSizeBytes: -1},
	} {
		if bad.valid() {
			t.Errorf("%s should be rejected", name)
		}
	}

	// Player names come from log lines, and a player picks their own name, so it
	// is the one field an outsider influences. It must be length-bounded.
	long := make([]byte, valheimMaxStr+1)
	for i := range long {
		long[i] = 'x'
	}
	longName := valheimStatus{Players: []string{string(long)}}
	if longName.valid() {
		t.Error("an over-long player name should be rejected")
	}
	longWorld := valheimStatus{World: string(long)}
	if longWorld.valid() {
		t.Error("an over-long world name should be rejected")
	}

	// Absurd list lengths bound memory.
	crowd := valheimStatus{Players: make([]string, valheimMaxPlayers+1)}
	if crowd.valid() {
		t.Error("too many players should be rejected")
	}
}

func TestValheimStoreDistinguishesUnknownFromStale(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	s := newValheimStore()

	// Never pushed is NOT the same as an empty server: without Known, a dead
	// collector would render identically to nobody being online.
	if got := s.snapshot(now); got.Known {
		t.Fatal("a store that has never been pushed to must report Known=false")
	}

	s.ingest(valheimStatus{Up: true, PlayerCount: 2, Players: []string{"Sigrid", "Bjorn"}}, now)
	got := s.snapshot(now)
	if !got.Known || got.Stale {
		t.Fatalf("a fresh push should be Known and not Stale, got %+v", got)
	}
	// Sorted on ingest so the render order does not follow awk's map iteration.
	if got.Players[0] != "Bjorn" {
		t.Errorf("players should be sorted, got %v", got.Players)
	}

	if got := s.snapshot(now.Add(valheimStaleAfter + time.Second)); !got.Stale {
		t.Error("a push older than valheimStaleAfter should be Stale")
	}
	if got := s.snapshot(now.Add(valheimStaleAfter + time.Second)); !got.Known {
		t.Error("stale is still Known; the distinction is the point")
	}
}
