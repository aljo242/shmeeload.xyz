package main

import (
	"sort"
	"sync"
	"time"
)

// Valheim status, pushed by foundry/valheim/pushstatus.sh. This replaced the
// Minecraft telemetry when that server was retired on 2026-08-13, and the shape
// is deliberately different because Valheim exposes much less:
//
//   - No player count over the wire. Valheim answers Steam A2S on SERVER_PORT+1
//     only when the server is public, and this one is private and password-only,
//     so presence is derived from the server log instead ("Got character ZDOID
//     from <name> : <id>", where 0:0 means the character left).
//   - No TPS. Valheim has no tick-rate telemetry; the concept does not exist.
//   - No in-game day number. Not logged.
//
// What it does have that Minecraft did not: the world seed and the mod list.
type valheimStatus struct {
	Up             bool     `json:"up"`
	Health         string   `json:"health"`
	UptimeSec      int64    `json:"uptimeSec"`
	Players        []string `json:"players"`
	PlayerCount    int      `json:"playerCount"`
	World          string   `json:"world"`
	Seed           string   `json:"seed"`
	WorldSizeBytes int64    `json:"worldSizeBytes"`
	Version        string   `json:"version"`
	Mods           []string `json:"mods"`
	BackupAgeH     float64  `json:"backupAgeH"`
	BackupCount    int      `json:"backupCount"`
}

const (
	valheimStaleAfter = 150 * time.Second // pushes are every 60s; ~2.5 missed
	valheimMaxPlayers = 64                // generous: the game caps far lower
	valheimMaxMods    = 64
	valheimMaxStr     = 64
)

// valid bounds a malformed or hostile push rather than policing values. The
// string caps matter most: these land in HTML, and an unbounded name from a log
// line is the one field an outsider could influence (by choosing a character
// name), so it is length-limited here and escaped at render time.
func (v *valheimStatus) valid() bool {
	if v.UptimeSec < 0 || v.PlayerCount < 0 || v.BackupCount < 0 || v.BackupAgeH < 0 {
		return false
	}
	if v.WorldSizeBytes < 0 {
		return false
	}
	if len(v.Players) > valheimMaxPlayers || len(v.Mods) > valheimMaxMods {
		return false
	}
	if len(v.World) > valheimMaxStr || len(v.Seed) > valheimMaxStr ||
		len(v.Version) > valheimMaxStr || len(v.Health) > valheimMaxStr {
		return false
	}
	for _, p := range v.Players {
		if len(p) > valheimMaxStr {
			return false
		}
	}
	for _, m := range v.Mods {
		if len(m) > valheimMaxStr {
			return false
		}
	}
	return true
}

type valheimStore struct {
	mu       sync.RWMutex
	val      valheimStatus
	received time.Time
}

func newValheimStore() *valheimStore { return &valheimStore{} }

func (s *valheimStore) ingest(v valheimStatus, now time.Time) {
	// Sort so the rendered order is stable rather than following whatever order
	// the collector's awk happened to emit.
	sort.Strings(v.Players)
	sort.Strings(v.Mods)
	s.mu.Lock()
	s.val, s.received = v, now
	s.mu.Unlock()
}

// valheimResponse is what /gamers/valheim serves. Reported distinguishes "never
// pushed" from "pushed a while ago": without it a dead collector looks exactly
// like an empty server, which is the failure mode worth being able to see.
type valheimResponse struct {
	valheimStatus
	Reported int64 `json:"reported"`
	Stale    bool  `json:"stale"`
	Known    bool  `json:"known"`
}

func (s *valheimStore) snapshot(now time.Time) valheimResponse {
	s.mu.RLock()
	v, recv := s.val, s.received
	s.mu.RUnlock()
	if recv.IsZero() {
		return valheimResponse{Known: false}
	}
	return valheimResponse{
		valheimStatus: v,
		Reported:      recv.Unix(),
		Stale:         now.Sub(recv) > valheimStaleAfter,
		Known:         true,
	}
}
