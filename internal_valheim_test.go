package main

import (
	"strings"
	"testing"
	"time"
)

// The dashboard's game panel switched from Minecraft to Valheim on 2026-08-13.
// This renders the real template rather than inspecting the view struct, because
// the failure worth catching is a template field name that no longer exists:
// html/template resolves those at execute time, so a rename silently produces a
// blank panel instead of a compile error.
func TestInternalDashboardRendersValheimPanel(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 13, 14, 0, 0, 0, time.UTC)

	store := newValheimStore()
	store.ingest(valheimStatus{
		Up:          true,
		PlayerCount: 2,
		Players:     []string{"Sigrid", "Bjorn"},
		World:       "shmeeland",
		Seed:        "U99eHjhrJ7",
		Version:     "5.4.2333",
	}, now)

	v := buildInternalView(piholeStats{}, store.snapshot(now), nil,
		func(string) []float64 { return nil }, 58, true, edgeReport{}, time.Time{}, now)

	var sb strings.Builder
	if err := internalTmpl.Execute(&sb, v); err != nil {
		t.Fatalf("template failed to execute: %v", err)
	}
	out := sb.String()

	for _, want := range []string{"Valheim", "shmeeland", "U99eHjhrJ7", "5.4.2333"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered dashboard is missing %q", want)
		}
	}
	// The retired panel must be gone, not merely empty.
	for _, gone := range []string{"Minecraft", "TPS"} {
		if strings.Contains(out, gone) {
			t.Errorf("rendered dashboard still contains retired %q", gone)
		}
	}
}

// A collector that has stopped reporting must not render as an idle server. This
// is the distinction the Known flag exists for, and it is easy to lose in a
// template that treats zero and unknown the same way.
func TestInternalDashboardShowsUnknownWhenNeverPushed(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 13, 14, 0, 0, 0, time.UTC)

	v := buildInternalView(piholeStats{}, newValheimStore().snapshot(now), nil,
		func(string) []float64 { return nil }, 58, true, edgeReport{}, time.Time{}, now)

	if v.VHPlayers != "?" {
		t.Errorf("a never-pushed collector should render players as %q, got %q", "?", v.VHPlayers)
	}
	if v.VHUp {
		t.Error("a never-pushed collector must not render the server as up")
	}
}
