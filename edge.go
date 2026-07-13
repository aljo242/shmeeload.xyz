package main

import (
	"sync"
	"time"
)

// edgeRoute mirrors one route entry edgelord pushes: request count this window,
// status counts by class (index 2=2xx, 4=4xx, 5=5xx), latency percentiles, and
// seconds since the last OK / last 5xx for that backend (-1 = none).
type edgeRoute struct {
	Host       string  `json:"host"`
	Reqs       int     `json:"reqs"`
	Status     [6]int  `json:"status"`
	P50ms      float64 `json:"p50ms"`
	P95ms      float64 `json:"p95ms"`
	LastOKSec  int     `json:"lastOkSec"`
	LastErrSec int     `json:"lastErrSec"`
}

type edgeCert struct {
	Host     string `json:"host"`
	DaysLeft int    `json:"daysLeft"`
}

// edgeReport is what edgelord POSTs to /internal/edge each interval.
type edgeReport struct {
	UptimeSec int         `json:"uptimeSec"`
	Routes    []edgeRoute `json:"routes"`
	Certs     []edgeCert  `json:"certs"`
}

const (
	edgeMaxRoutes  = 32
	edgeStaleAfter = 3 * time.Minute // edgelord pushes every ~60s
)

// valid bounds sizes so a malformed or hostile push can't balloon memory.
func (r *edgeReport) valid() bool {
	if len(r.Routes) > edgeMaxRoutes || len(r.Certs) > edgeMaxRoutes {
		return false
	}
	for _, rt := range r.Routes {
		if len(rt.Host) > 64 {
			return false
		}
	}
	for _, c := range r.Certs {
		if len(c.Host) > 64 {
			return false
		}
	}
	return true
}

// edgeStore holds the latest report from edgelord.
type edgeStore struct {
	mu       sync.RWMutex
	rep      edgeReport
	received time.Time
}

func newEdgeStore() *edgeStore { return &edgeStore{} }

func (s *edgeStore) ingest(r edgeReport, now time.Time) {
	s.mu.Lock()
	s.rep, s.received = r, now
	s.mu.Unlock()
}

func (s *edgeStore) get() (edgeReport, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rep, s.received
}
