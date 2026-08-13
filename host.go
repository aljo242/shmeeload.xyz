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

// hostNet is one wired interface's link state. This exists because the disk and
// service checks say nothing about the physical layer: a cable degrading to
// 100M, or one quietly accumulating errors, is invisible until something feels
// slow.
type hostNet struct {
	Name        string `json:"name"`
	SpeedMbs    int    `json:"speedMbs"`
	Duplex      string `json:"duplex"`
	Carrier     bool   `json:"carrier"`
	Errs        int64  `json:"errs"`        // rx+tx errors; dropped is deliberately excluded
	CarrierErrs int64  `json:"carrierErrs"` // link flaps
}

// hostDrive is one physical drive's SMART state. Present is its own field
// because a USB-attached drive falling off the bus is the failure this is here
// to catch, and an absent drive reports no other attribute at all.
type hostDrive struct {
	Dev      string  `json:"dev"`
	Model    string  `json:"model"`
	Present  bool    `json:"present"`
	Health   string  `json:"health"` // PASSED, FAILED, or "" when unreadable
	TempC    float64 `json:"tempC"`
	Realloc  int     `json:"realloc"` // Reallocated_Sector_Ct
	Pending  int     `json:"pending"` // Current_Pending_Sector
	Uncorr   int     `json:"uncorr"`  // Offline_Uncorrectable
	SelfTest string  `json:"selfTest"`
}

// hostBackup is the freshness of one backup tier, reported by the box that owns
// the drives.
//
// MaxAgeH travels WITH the reading instead of being a constant here, because the
// tiers run on schedules that differ by three orders of magnitude: snapshots four
// times a day, the config pull nightly, the cold-drive rotation monthly. A single
// threshold cannot be both "the 6-hourly job missed a run" and "the monthly job is
// fine at three weeks", and picking one would either cry wolf about the rotation
// or stay silent for a month about the snapshots.
//
// Count matters as much as age. A tier that runs successfully and produces nothing
// is the failure this exists to catch: the unit does not fail, so failedUnits stays
// clean, and the age of a snapshot that was never taken is not a number anyone sees.
type hostBackup struct {
	Name    string  `json:"name"`
	AgeH    float64 `json:"ageH"`    // age of the newest snapshot
	Count   int     `json:"count"`   // how many are retained
	MaxAgeH float64 `json:"maxAgeH"` // older than this is a problem, set by the collector
}

// switchPort is one managed-switch port. Reported by whichever box can reach the
// switch (only foundry does), not by the switch itself, which speaks no
// protocol we can push from.
type switchPort struct {
	Port  int    `json:"port"`
	Link  string `json:"link"` // "1000M Full", "Link Down", ...
	TxBad int64  `json:"txBad"`
	RxBad int64  `json:"rxBad"`
}

// hostDNS is one name-resolution assertion. Split-horizon has broken silently
// before (Pi-hole v6 regenerating custom.list), and nothing noticed until
// someone tried to load the site.
type hostDNS struct {
	Name string `json:"name"`
	Want string `json:"want"`
	Got  string `json:"got"`
	OK   bool   `json:"ok"`
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
	// The physical tier. All optional: a box that does not report a section
	// renders nothing for it rather than showing a false negative.
	Net    []hostNet    `json:"net,omitempty"`
	Drives []hostDrive  `json:"drives,omitempty"`
	Switch []switchPort `json:"switch,omitempty"`
	DNS    []hostDNS    `json:"dns,omitempty"`
	// Backup tiers. Optional for the same reason as the rest: a box that does
	// not own the drives reports nothing here rather than a false all-clear.
	Backups []hostBackup `json:"backups,omitempty"`
}

const (
	hostMaxDisks    = 16
	hostMaxServices = 32
	hostMaxNet      = 8
	hostMaxDrives   = 8
	hostMaxPorts    = 52 // a 48-port switch plus uplinks, well above the 16 here
	hostMaxDNS      = 8
	hostMaxBackups  = 8
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
	if len(r.Net) > hostMaxNet || len(r.Drives) > hostMaxDrives ||
		len(r.Switch) > hostMaxPorts || len(r.DNS) > hostMaxDNS ||
		len(r.Backups) > hostMaxBackups {
		return false
	}
	for _, b := range r.Backups {
		// Negative values would invert every comparison downstream and render a
		// stale tier as healthy, which is the one failure this must not produce.
		if len(b.Name) > 24 || b.AgeH < 0 || b.Count < 0 || b.MaxAgeH < 0 {
			return false
		}
	}
	for _, d := range r.Disks {
		if len(d.Mount) > 48 {
			return false
		}
	}
	for _, n := range r.Net {
		if len(n.Name) > 32 || len(n.Duplex) > 8 {
			return false
		}
	}
	for _, d := range r.Drives {
		if len(d.Dev) > 16 || len(d.Model) > 48 || len(d.Health) > 16 || len(d.SelfTest) > 64 {
			return false
		}
	}
	for _, p := range r.Switch {
		if len(p.Link) > 24 {
			return false
		}
	}
	for _, d := range r.DNS {
		if len(d.Name) > 64 || len(d.Want) > 64 || len(d.Got) > 64 {
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
