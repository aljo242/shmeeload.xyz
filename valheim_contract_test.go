package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// The exact JSON foundry/valheim/pushstatus.sh emitted against the live server.
const liveCollectorPayload = `{"up": true, "health": "none", "uptimeSec": 1028, "players": [], "playerCount": 0, "world": "shmeeland", "seed": "U99eHjhrJ7", "worldSizeBytes": 21906346, "version": "5.4.2333", "mods": ["Valheim.ServersideQoL"], "backupAgeH": 0.3, "backupCount": 6}`

func TestValheimAcceptsTheRealCollectorPayload(t *testing.T) {
	var v valheimStatus
	dec := json.NewDecoder(io.LimitReader(strings.NewReader(liveCollectorPayload), 16<<10))
	dec.DisallowUnknownFields() // the route uses this: any field drift fails here
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("collector payload did not decode: %v", err)
	}
	if !v.valid() {
		t.Fatal("collector payload failed validation")
	}
	if v.Seed != "U99eHjhrJ7" || v.World != "shmeeland" || v.Version != "5.4.2333" {
		t.Fatalf("fields did not map: %+v", v)
	}
	s := newValheimStore()
	s.ingest(v, time.Now())
	got := s.snapshot(time.Now())
	if !got.Known || got.Stale || got.Seed != "U99eHjhrJ7" {
		t.Fatalf("round trip lost data: %+v", got)
	}
}
