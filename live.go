package main

import (
	"sync"
	"time"
)

// pushedStatus is the live server telemetry the game host POSTs to /gamers/live.
// It carries what Server List Ping cannot: tick rate, in-game day, process
// uptime, whitelist size, and a free-form service-health map for the status
// page.
type pushedStatus struct {
	TPS       float64           `json:"tps"`
	MsPerTick float64           `json:"msPerTick"`
	Day       int               `json:"day"`
	UptimeSec int64             `json:"uptimeSec"`
	Whitelist int               `json:"whitelist"`
	Services  map[string]string `json:"services"`
}

// liveMaxServices and the length caps bound what an ingest payload can hold, so
// a malformed or hostile push cannot balloon memory.
const (
	liveMaxServices   = 24
	liveMaxServiceKey = 32
	liveMaxServiceVal = 32
	liveStaleAfter    = 150 * time.Second // ~2.5 missed 60s pushes
)

// valid reports whether the payload is in sane ranges. It also trims the
// services map to the caps. It returns false for values that indicate a broken
// or hostile sender rather than clamping them silently.
func (p *pushedStatus) valid() bool {
	if p.TPS < 0 || p.TPS > 100 {
		return false
	}
	if p.MsPerTick < 0 || p.MsPerTick > 100000 {
		return false
	}
	if p.Day < 0 || p.UptimeSec < 0 || p.Whitelist < 0 {
		return false
	}
	if len(p.Services) > liveMaxServices {
		return false
	}
	for k, v := range p.Services {
		if len(k) > liveMaxServiceKey || len(v) > liveMaxServiceVal {
			return false
		}
	}
	return true
}

// liveStore holds the most recent pushed telemetry and when it arrived, so page
// loads read a cached snapshot instead of reaching the game host directly.
type liveStore struct {
	mu       sync.RWMutex
	val      pushedStatus
	received time.Time
}

func newLiveStore() *liveStore { return &liveStore{} }

func (s *liveStore) ingest(p pushedStatus, now time.Time) {
	s.mu.Lock()
	s.val = p
	s.received = now
	s.mu.Unlock()
}

func (s *liveStore) snapshot() (pushedStatus, time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.val, s.received, !s.received.IsZero()
}

// liveResponse is the merged view served at GET /gamers/live: the always-fresh
// Server List Ping fields plus the pushed telemetry (nil when nothing has been
// pushed yet, so the page shows a dash rather than a fake zero).
type liveResponse struct {
	Up      bool       `json:"up"`
	Online  int        `json:"online"`
	Max     int        `json:"max"`
	Players []mcPlayer `json:"players"`

	TPS       *float64          `json:"tps,omitempty"`
	MsPerTick *float64          `json:"msPerTick,omitempty"`
	Day       *int              `json:"day,omitempty"`
	UptimeSec *int64            `json:"uptimeSec,omitempty"`
	Whitelist *int              `json:"whitelist,omitempty"`
	Services  map[string]string `json:"services,omitempty"`

	Stale    bool  `json:"stale"`    // pushed telemetry older than liveStaleAfter
	PushedAt int64 `json:"pushedAt"` // unix seconds of the last push, 0 if never
}

// buildLive merges the SLP status with the pushed telemetry and the site's own
// service checks as of now. siteServices (e.g. Pi-hole) are merged over the
// pushed service map into a fresh map, so the stored push is never mutated.
func buildLive(slp mcStatus, s *liveStore, siteServices map[string]string, now time.Time) liveResponse {
	r := liveResponse{
		Up:      slp.Up,
		Online:  slp.Online,
		Max:     slp.Max,
		Players: slp.Players,
	}
	services := map[string]string{}
	if p, received, ok := s.snapshot(); ok {
		tps, mspt, day, up, wl := p.TPS, p.MsPerTick, p.Day, p.UptimeSec, p.Whitelist
		r.TPS = &tps
		r.MsPerTick = &mspt
		r.Day = &day
		r.UptimeSec = &up
		r.Whitelist = &wl
		r.PushedAt = received.Unix()
		r.Stale = now.Sub(received) > liveStaleAfter
		for k, v := range p.Services {
			services[k] = v
		}
	}
	for k, v := range siteServices {
		services[k] = v
	}
	if len(services) > 0 {
		r.Services = services
	}
	return r
}
