package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The failure this whole feature exists to prevent is a backup tier that reads
// healthy while being broken, so these tests are mostly about what must NOT come
// back sevOK.
func TestBackupTierSev(t *testing.T) {
	t.Parallel()

	// A tier that ran, exited 0, and produced nothing. failedUnits stays clean
	// and there is no age to look stale, so without this case it renders as fine
	// right up until someone needs a restore.
	if s, msg := backupTierSev(hostBackup{Name: "nas-history", Count: 0, MaxAgeH: 8}); s != sevBad {
		t.Errorf("zero snapshots must be bad, got %v (%q)", s, msg)
	}

	// Zero count outranks a healthy-looking age. An empty tier whose newest entry
	// is somehow recent is still an empty tier.
	if s, _ := backupTierSev(hostBackup{Name: "x", Count: 0, AgeH: 0, MaxAgeH: 24}); s != sevBad {
		t.Errorf("zero count must be bad even at age 0, got %v", s)
	}

	for name, tc := range map[string]struct {
		in   hostBackup
		want sev
	}{
		// Each tier is graded against its own schedule. The same 30-hour age is
		// fine for a monthly rotation and a missed cycle for a 6-hourly one,
		// which is the entire reason MaxAgeH travels with the reading.
		"6-hourly, fresh":      {hostBackup{Count: 4, AgeH: 2, MaxAgeH: 8}, sevOK},
		"6-hourly, late":       {hostBackup{Count: 4, AgeH: 9, MaxAgeH: 8}, sevWarn},
		"6-hourly, missed":     {hostBackup{Count: 4, AgeH: 30, MaxAgeH: 8}, sevBad},
		"nightly, fresh":       {hostBackup{Count: 30, AgeH: 10, MaxAgeH: 30}, sevOK},
		"nightly, late":        {hostBackup{Count: 30, AgeH: 31, MaxAgeH: 30}, sevWarn},
		"monthly, 30h is fine": {hostBackup{Count: 1, AgeH: 30, MaxAgeH: 840}, sevOK},
		"monthly, overdue":     {hostBackup{Count: 1, AgeH: 900, MaxAgeH: 840}, sevWarn},

		// A collector that declares no schedule gets no opinion, rather than a
		// default threshold that would be wrong for some tier.
		"no schedule declared": {hostBackup{Count: 3, AgeH: 5000, MaxAgeH: 0}, sevOK},
	} {
		if got, _ := backupTierSev(tc.in); got != tc.want {
			t.Errorf("%s: want %v, got %v", name, tc.want, got)
		}
	}
}

// A malformed push must not be able to make a stale tier look healthy. Negative
// values would invert the comparisons and do exactly that.
func TestHostReportRejectsBadBackups(t *testing.T) {
	t.Parallel()

	ok := hostReport{Box: "foundry", Backups: []hostBackup{{Name: "nas-history", AgeH: 2, Count: 4, MaxAgeH: 8}}}
	if !ok.valid() {
		t.Fatal("a normal backup section should validate")
	}

	for name, b := range map[string]hostBackup{
		"negative age":   {Name: "x", AgeH: -1},
		"negative count": {Name: "x", Count: -1},
		"negative max":   {Name: "x", MaxAgeH: -1},
		"long name":      {Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	} {
		r := hostReport{Box: "foundry", Backups: []hostBackup{b}}
		if r.valid() {
			t.Errorf("%s should be rejected", name)
		}
	}

	// Bounded like every other section, so a hostile push cannot balloon memory.
	many := make([]hostBackup, hostMaxBackups+1)
	for i := range many {
		many[i] = hostBackup{Name: "x", MaxAgeH: 8}
	}
	tooMany := hostReport{Box: "foundry", Backups: many}
	if tooMany.valid() {
		t.Error("more than hostMaxBackups should be rejected")
	}
}

// The template renders field names at EXECUTE time, so a renamed or misspelled
// field produces a silently blank cell rather than a compile error. This whole
// change replaced a single .Backup cell with a range over .Backups, which is
// exactly the shape that fails quietly, so assert the tiers reach the HTML.
func TestInternalPageRendersBackupTiers(t *testing.T) {
	t.Parallel()

	hosts := []hostEntry{{
		rep: hostReport{
			Box: "foundry",
			Backups: []hostBackup{
				{Name: "history", AgeH: 1.9, Count: 4, MaxAgeH: 8},
				{Name: "config", AgeH: 12.8, Count: 2, MaxAgeH: 30},
				{Name: "stalled", AgeH: 99, Count: 0, MaxAgeH: 8},
			},
		},
		received: time.Now(),
	}}
	v := buildInternalView(piholeStats{}, valheimResponse{}, hosts,
		func(string) []float64 { return nil }, 40, true, edgeReport{}, time.Time{}, time.Now())

	rec := httptest.NewRecorder()
	renderInternal(rec, v)
	body := rec.Body.String()

	for _, want := range []string{"history", "config", "4 kept", "2 kept"} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered page is missing %q; the template field name is probably wrong", want)
		}
	}
	// A tier with nothing in it must say so rather than showing an age of zero,
	// which would read as "just ran".
	if !strings.Contains(body, "never") {
		t.Error("a zero-count tier should render as 'never', not as a fresh age")
	}
	// And it has to reach the warnings, not just the panel.
	if !strings.Contains(body, "no snapshots at all") {
		t.Error("a zero-count tier should raise a visible warning")
	}
}
