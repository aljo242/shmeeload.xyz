package main

import (
	"testing"
	"time"
)

func TestBuildEdgePanel(t *testing.T) {
	now := time.Now()

	// Never reported -> no panel.
	if p, _, _ := buildEdgePanel(edgeReport{}, time.Time{}, now); p != nil {
		t.Error("expected nil panel when edgelord has never reported")
	}

	rep := edgeReport{
		UptimeSec: 3600,
		Routes: []edgeRoute{
			// healthy: last OK more recent than last err
			{Host: "site", Reqs: 100, Status: [6]int{0, 0, 99, 0, 1, 0}, LastOKSec: 1, LastErrSec: -1},
			// backend down: last err (5s) more recent than last OK (40s)
			{Host: "map", Reqs: 20, Status: [6]int{0, 0, 10, 0, 0, 10}, LastOKSec: 40, LastErrSec: 5},
		},
		Certs: []edgeCert{{Host: "site", DaysLeft: 3}},
	}
	p, warns, bads := buildEdgePanel(rep, now.Add(-30*time.Second), now)
	if p == nil {
		t.Fatal("expected a panel")
	}
	if len(p.Routes) != 2 {
		t.Fatalf("routes = %d, want 2", len(p.Routes))
	}
	// routes are sorted by host: "map" then "site"
	if p.Routes[0].Host != "map" || p.Routes[0].BackendUp {
		t.Errorf("map should be present and backend down, got %+v", p.Routes[0])
	}
	if p.Routes[1].Host != "site" || !p.Routes[1].BackendUp {
		t.Errorf("site should be present and backend up, got %+v", p.Routes[1])
	}
	// map's 10 5xx out of 120 total = ~8.3% -> a critical
	foundErr, foundDown, foundCert := false, false, false
	for _, b := range bads {
		if b == "edge: map backend down" {
			foundDown = true
		}
		if len(b) > 10 && b[:11] == "edge: 8.3% " {
			foundErr = true
		}
	}
	if !foundDown {
		t.Errorf("expected a 'map backend down' critical, got %v", bads)
	}
	if !foundErr {
		t.Errorf("expected an error-rate critical, got %v", bads)
	}
	// cert at 3 days -> bad class present
	for _, c := range p.Certs {
		if c.Host == "site" && c.Class == sevBad.class() {
			foundCert = true
		}
	}
	if !foundCert {
		t.Errorf("cert at 3 days should be class 'bad', got %+v", p.Certs)
	}
	_ = warns
}
