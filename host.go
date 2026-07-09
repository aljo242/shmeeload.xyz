package main

import (
	"sort"
	"sync"
	"time"
)

// hostDisk is one filesystem's usage on a box.
type hostDisk struct {
	Mount   string  `json:"mount"`
	UsedPct int     `json:"usedPct"`
	UsedGB  float64 `json:"usedGB"`
	TotalGB float64 `json:"totalGB"`
}

// hostReport is what each box (foundry, the Pi) POSTs to /internal/host. Fields
// left at zero render as "--"; BackupAgeH > 0 marks a box that also reports a
// Minecraft backup (foundry).
type hostReport struct {
	Box         string            `json:"box"`
	UptimeSec   int64             `json:"uptimeSec"`
	Load1       float64           `json:"load1"`
	Cores       int               `json:"cores"`
	MemUsedMB   int               `json:"memUsedMB"`
	MemTotalMB  int               `json:"memTotalMB"`
	SwapUsedMB  int               `json:"swapUsedMB"`
	SwapTotalMB int               `json:"swapTotalMB"`
	TempC       float64           `json:"tempC"`
	FailedUnits int               `json:"failedUnits"`
	Updates     int               `json:"updates"`
	RebootReq   bool              `json:"rebootRequired"`
	CIBuilding  bool              `json:"ciBuilding"`
	Disks       []hostDisk        `json:"disks"`
	Services    map[string]string `json:"services"`
	BackupAgeH  float64           `json:"backupAgeH"`
	BackupGB    float64           `json:"backupGB"`
}

const (
	hostMaxDisks    = 16
	hostMaxServices = 32
	hostStaleAfter  = 5 * time.Minute // boxes push every ~2 min
)

// valid keeps a malformed or hostile push from ballooning memory. Numeric
// ranges are lenient; the point is bounding sizes, not policing values.
func (r *hostReport) valid() bool {
	if r.Box == "" || len(r.Box) > 24 {
		return false
	}
	if len(r.Disks) > hostMaxDisks || len(r.Services) > hostMaxServices {
		return false
	}
	for _, d := range r.Disks {
		if len(d.Mount) > 48 {
			return false
		}
	}
	for k, v := range r.Services {
		if len(k) > 32 || len(v) > 24 {
			return false
		}
	}
	return true
}

type hostEntry struct {
	rep      hostReport
	received time.Time
}

// hostStore holds the latest report per box.
type hostStore struct {
	mu sync.RWMutex
	m  map[string]hostEntry
}

func newHostStore() *hostStore { return &hostStore{m: map[string]hostEntry{}} }

func (s *hostStore) ingest(r hostReport, now time.Time) {
	s.mu.Lock()
	s.m[r.Box] = hostEntry{rep: r, received: now}
	s.mu.Unlock()
}

// all returns the reports sorted by box name for a stable render order.
func (s *hostStore) all() []hostEntry {
	s.mu.RLock()
	out := make([]hostEntry, 0, len(s.m))
	for _, e := range s.m {
		out = append(out, e)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].rep.Box < out[j].rep.Box })
	return out
}
