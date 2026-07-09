package main

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The internal homelab page is rendered server-side and refreshed by a plain
// <meta refresh>. Page navigations carry Basic Auth reliably (unlike fetch), so
// this needs no cookie, token, or client script.

// sev is a metric's health: ok, warn, or bad. class() is the CSS suffix.
type sev int

const (
	sevOK sev = iota
	sevWarn
	sevBad
)

func (s sev) class() string {
	switch s {
	case sevBad:
		return "bad"
	case sevWarn:
		return "warn"
	default:
		return ""
	}
}

func diskSev(pct int) sev {
	switch {
	case pct >= 90:
		return sevBad
	case pct >= 80:
		return sevWarn
	default:
		return sevOK
	}
}

func tempSev(c float64) sev {
	switch {
	case c >= 80:
		return sevBad
	case c >= 68:
		return sevWarn
	default:
		return sevOK
	}
}

func pctSev(used, total, warn, bad int) sev {
	if total <= 0 {
		return sevOK
	}
	p := used * 100 / total
	switch {
	case p >= bad:
		return sevBad
	case p >= warn:
		return sevWarn
	default:
		return sevOK
	}
}

func loadSev(load float64, cores int) sev {
	if cores <= 0 {
		return sevOK // unknown core count; don't judge load
	}
	switch r := load / float64(cores); {
	case r >= 3:
		return sevBad
	case r >= 1.5:
		return sevWarn
	default:
		return sevOK
	}
}

func backupSev(ageH float64) sev {
	switch {
	case ageH >= 48:
		return sevBad
	case ageH >= 26:
		return sevWarn
	default:
		return sevOK
	}
}

func serviceUp(state string) bool {
	switch state {
	case "up", "idle", "active", "running":
		return true
	default:
		return false
	}
}

type internalRow struct{ Count, Domain string }

type hostDiskView struct {
	Mount, Used, BarClass string
	Pct                   int
}

type hostServiceView struct {
	Name, State string
	Up          bool
}

type hostPanel struct {
	Box, Reported                             string
	Up, Stale                                 bool
	Uptime, Load, Mem, Swap, Temp             string
	LoadClass, MemClass, SwapClass, TempClass string
	FailedUnits, FailedClass                  string
	Updates                                   string
	RebootReq, CIBuilding                     bool
	Backup, BackupClass                       string
	LoadSpark, DiskSpark                      template.HTML
	Disks                                     []hostDiskView
	Services                                  []hostServiceView
}

type internalView struct {
	StatusClass, StatusText                   string
	Warnings, Criticals                       []string
	CertDays, CertClass                       string
	PiUp                                      bool
	Queries, Blocked, Pct, Blocklist, Clients string
	Cache, Blocking, BlockingClass            string
	PiSpark                                   template.HTML
	Top, TopClients                           []internalRow
	MCUp                                      bool
	Online, TPS, Day                          string
	Hosts                                     []hostPanel
	Checked                                   string
}

var internalTmpl = template.Must(template.New("internal").Parse(internalHTML))

// commafy renders an int with thousands separators.
func commafy(n int) string {
	neg := n < 0
	s := strconv.Itoa(n)
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(s[i])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func fmtDuration(sec int64) string {
	d, h, m := sec/86400, (sec%86400)/3600, (sec%3600)/60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// sparkline renders a series as a tiny inline SVG polyline over a 100x20 box.
// The markup is server-generated from floats, so template.HTML is safe here.
func sparkline(points []float64) template.HTML {
	if len(points) < 2 {
		return ""
	}
	lo, hi := points[0], points[0]
	for _, v := range points {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	span := hi - lo
	if span == 0 {
		span = 1
	}
	var b strings.Builder
	for i, v := range points {
		x := 100 * float64(i) / float64(len(points)-1)
		y := 20 - (v-lo)/span*20
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%.1f,%.1f", x, y)
	}
	return template.HTML(fmt.Sprintf( //nolint:gosec // markup built from floats, no user input
		`<svg class="spark" viewBox="0 0 100 20" preserveAspectRatio="none"><polyline points="%s"/></svg>`, b.String()))
}

func buildHostPanel(e hostEntry, now time.Time) (hostPanel, []string, []string) {
	r := e.rep
	stale := now.Sub(e.received) > hostStaleAfter
	p := hostPanel{Box: r.Box, Reported: e.received.Format("15:04:05"), Up: !stale, Stale: stale, Uptime: fmtDuration(r.UptimeSec)}
	var warns, bads []string
	note := func(s sev, msg string) {
		switch s {
		case sevWarn:
			warns = append(warns, r.Box+": "+msg)
		case sevBad:
			bads = append(bads, r.Box+": "+msg)
		}
	}

	ls := loadSev(r.Load1, r.Cores)
	p.Load, p.LoadClass = strconv.FormatFloat(r.Load1, 'f', 2, 64), ls.class()
	note(ls, fmt.Sprintf("load %.2f", r.Load1))

	if r.MemTotalMB > 0 {
		p.Mem = fmt.Sprintf("%.1f / %.0f GB", float64(r.MemUsedMB)/1024, float64(r.MemTotalMB)/1024)
		ms := pctSev(r.MemUsedMB, r.MemTotalMB, 82, 92)
		p.MemClass = ms.class()
		note(ms, fmt.Sprintf("memory %d%%", r.MemUsedMB*100/r.MemTotalMB))
	} else {
		p.Mem = "--"
	}

	if r.SwapTotalMB > 0 {
		p.Swap = fmt.Sprintf("%.1f / %.0f GB", float64(r.SwapUsedMB)/1024, float64(r.SwapTotalMB)/1024)
		ss := pctSev(r.SwapUsedMB, r.SwapTotalMB, 50, 80)
		p.SwapClass = ss.class()
		note(ss, fmt.Sprintf("swap %d%%", r.SwapUsedMB*100/r.SwapTotalMB))
	} else {
		p.Swap = "none"
	}

	if r.TempC > 0 {
		p.Temp = fmt.Sprintf("%.0f°C", r.TempC)
		tsv := tempSev(r.TempC)
		p.TempClass = tsv.class()
		note(tsv, fmt.Sprintf("temp %.0f°C", r.TempC))
	} else {
		p.Temp = "--"
	}

	if r.FailedUnits > 0 {
		p.FailedUnits, p.FailedClass = strconv.Itoa(r.FailedUnits), sevWarn.class()
		note(sevWarn, fmt.Sprintf("%d failed unit(s)", r.FailedUnits))
	}

	if r.Updates > 0 {
		p.Updates = strconv.Itoa(r.Updates)
	}
	p.RebootReq, p.CIBuilding = r.RebootReq, r.CIBuilding
	if r.RebootReq {
		warns = append(warns, r.Box+": reboot required")
	}

	for _, d := range r.Disks {
		ds := diskSev(d.UsedPct)
		p.Disks = append(p.Disks, hostDiskView{
			Mount: d.Mount, Pct: d.UsedPct, BarClass: ds.class(),
			Used: fmt.Sprintf("%.0f/%.0f GB", d.UsedGB, d.TotalGB),
		})
		note(ds, fmt.Sprintf("disk %s %d%%", d.Mount, d.UsedPct))
	}

	keys := make([]string, 0, len(r.Services))
	for k := range r.Services {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		up := serviceUp(r.Services[k])
		p.Services = append(p.Services, hostServiceView{Name: k, State: r.Services[k], Up: up})
		if !up {
			bads = append(bads, r.Box+": "+k+" down")
		}
	}

	if r.BackupAgeH > 0 {
		bs := backupSev(r.BackupAgeH)
		p.Backup, p.BackupClass = fmt.Sprintf("%s · %.1f GB", fmtDuration(int64(r.BackupAgeH*3600)), r.BackupGB), bs.class()
		note(bs, fmt.Sprintf("mc backup %s old", fmtDuration(int64(r.BackupAgeH*3600))))
	}

	if stale {
		warns = append(warns, r.Box+": stale (no recent report)")
	}
	return p, warns, bads
}

func buildInternalView(pi piholeStats, mc liveResponse, hosts []hostEntry, hist func(string) []float64, certDays int, certOK bool, now time.Time) internalView {
	v := internalView{PiUp: pi.Up, MCUp: mc.Up}
	v.PiSpark = sparkline(hist("pihole.blocked_pct"))
	var warns, bads []string

	if certOK {
		v.CertDays = strconv.Itoa(certDays) + "d"
		cs := sevOK
		switch {
		case certDays <= 5:
			cs = sevBad
		case certDays <= 14:
			cs = sevWarn
		}
		v.CertClass = cs.class()
		if msg := fmt.Sprintf("TLS cert expires in %dd", certDays); cs == sevBad {
			bads = append(bads, msg)
		} else if cs == sevWarn {
			warns = append(warns, msg)
		}
	} else {
		v.CertDays = "--"
	}

	if pi.Up {
		v.Queries = commafy(pi.QueriesToday)
		v.Blocked = commafy(pi.BlockedToday)
		v.Pct = strconv.FormatFloat(pi.PercentBlocked, 'f', 1, 64) + "%"
		v.Cache = strconv.FormatFloat(pi.CacheHitPct, 'f', 0, 64) + "%"
		v.Blocklist = commafy(pi.BlocklistSize)
		v.Clients = commafy(pi.ActiveClients)
		v.Blocking = pi.Blocking
		if pi.Blocking == "disabled" {
			v.BlockingClass = sevBad.class()
			bads = append(bads, "pi-hole: blocking DISABLED")
		}
		for _, d := range pi.TopBlocked {
			v.Top = append(v.Top, internalRow{Count: commafy(d.Count), Domain: d.Domain})
		}
		for _, cl := range pi.TopClients {
			name := cl.Name
			if name == "" {
				name = cl.IP
			}
			v.TopClients = append(v.TopClients, internalRow{Count: commafy(cl.Count), Domain: name})
		}
	} else {
		v.Queries, v.Blocked, v.Pct, v.Cache, v.Blocklist, v.Clients = "--", "--", "--", "--", "--", "--"
	}
	if pi.CheckedAt > 0 {
		v.Checked = "pi-hole checked " + time.Unix(pi.CheckedAt, 0).Format("15:04:05")
	} else {
		v.Checked = "pi-hole not yet reached"
	}

	if mc.Up {
		v.Online = strconv.Itoa(mc.Online) + "/" + strconv.Itoa(mc.Max)
	} else {
		v.Online = "--"
	}
	if mc.TPS != nil {
		v.TPS = strconv.FormatFloat(*mc.TPS, 'f', 1, 64)
	} else {
		v.TPS = "--"
	}
	if mc.Day != nil {
		v.Day = commafy(*mc.Day)
	} else {
		v.Day = "--"
	}

	for _, e := range hosts {
		pnl, w, b := buildHostPanel(e, now)
		pnl.LoadSpark = sparkline(hist(e.rep.Box + ".load"))
		pnl.DiskSpark = sparkline(hist(e.rep.Box + ".disk"))
		v.Hosts = append(v.Hosts, pnl)
		warns = append(warns, w...)
		bads = append(bads, b...)
	}
	if !pi.Up {
		bads = append(bads, "pi-hole: unreachable")
	}

	v.Warnings, v.Criticals = warns, bads
	switch {
	case len(bads) > 0:
		v.StatusClass = "bad"
		v.StatusText = fmt.Sprintf("%d critical, %d warning", len(bads), len(warns))
	case len(warns) > 0:
		v.StatusClass = "warn"
		v.StatusText = fmt.Sprintf("%d warning", len(warns))
	default:
		v.StatusClass = "ok"
		v.StatusText = "all systems nominal"
	}
	return v
}

func renderInternal(w http.ResponseWriter, v internalView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_ = internalTmpl.Execute(w, v)
}

const internalHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<meta http-equiv="refresh" content="60"/>
<meta name="robots" content="noindex, nofollow"/>
<meta name="theme-color" content="#0b0f14"/>
<title>homelab internal</title>
<link rel="shortcut icon" href="/static/img/1favicon.ico" type="image/x-icon"/>
<style>
html,body{margin:0;background:#0b0f14}
body{font-family:-apple-system,"Segoe UI",Roboto,Arial,sans-serif;color:#dfe6ee;min-height:100vh}
.wrap{max-width:760px;margin:0 auto;padding:20px 16px 48px}
h1{font-size:20px;font-weight:700;margin:0 0 2px;letter-spacing:.3px}
.muted{color:#7d8b99;font-size:13px}
.rollup{border-radius:10px;padding:12px 16px;margin:14px 0;font-weight:700;font-size:15px;border:1px solid #223041;background:#121821}
.rollup.ok{border-color:#1e6b3f;background:#0f1b14;color:#37d67a}
.rollup.warn{border-color:#7a5a1e;background:#1b160f;color:#ffc061}
.rollup.bad{border-color:#7a2626;background:#1b0f0f;color:#ff8b8b}
.rollup ul{margin:8px 0 0;padding-left:18px;font-weight:400;font-size:13px}
.rollup li.bad{color:#ff8b8b}.rollup li.warn{color:#ffc061}
.panel{background:#121821;border:1px solid #223041;border-radius:10px;padding:16px 18px;margin:16px 0}
.panel h2{font-size:14px;text-transform:uppercase;letter-spacing:1px;color:#8fb7ff;margin:0 0 12px;display:flex;align-items:center;gap:8px}
.dot{width:10px;height:10px;border-radius:50%;background:#556;display:inline-block}
.dot.sm{width:8px;height:8px}
.dot.up{background:#37d67a;box-shadow:0 0 8px #37d67a66}
.dot.down{background:#ff5a5a;box-shadow:0 0 8px #ff5a5a66}
.grid{display:flex;flex-wrap:wrap;gap:22px}
.stat .n{font-size:30px;font-weight:800;line-height:1.1}
.stat .l{font-size:12px;color:#7d8b99;text-transform:uppercase;letter-spacing:.5px}
.n.block{color:#ff9d3c}.n.pct{color:#37d67a}
.n.warn{color:#ffc061}.n.bad{color:#ff8b8b}
table{width:100%;border-collapse:collapse;margin-top:6px;font-size:14px}
td{padding:6px 4px;border-bottom:1px solid #1b2431}
td.c{text-align:right;color:#ff9d3c;font-variant-numeric:tabular-nums;width:72px;font-weight:700}
td.d{color:#cfe;word-break:break-all;font-family:ui-monospace,"SF Mono",Menlo,monospace;font-size:12.5px}
.disk{margin:9px 0}
.dlabel{font-size:12.5px;color:#aeb9c6;margin-bottom:3px}
.bar{height:8px;background:#0a0f16;border:1px solid #223041;border-radius:4px;overflow:hidden}
.fill{height:100%;background:#37d67a}
.fill.warn{background:#ff9d3c}.fill.bad{background:#ff5a5a}
.svcs{display:flex;flex-wrap:wrap;gap:14px;margin-top:12px}
.svc{font-size:12.5px;color:#cfe;display:flex;align-items:center;gap:5px}
.spark{display:block;width:100%;height:26px}
.spark polyline{fill:none;stroke:#8fb7ff;stroke-width:1.4;vector-effect:non-scaling-stroke}
.sparks{display:flex;gap:18px;margin-top:12px;flex-wrap:wrap}
.sparkbox{flex:1;min-width:150px}
.sparklbl{font-size:11px;color:#7d8b99;text-transform:uppercase;letter-spacing:.5px;margin-bottom:3px}
.cert.warn{color:#ffc061}.cert.bad{color:#ff8b8b}
.tag{font-size:11px;font-weight:600;color:#37d67a;background:#0f1b14;border:1px solid #1e6b3f;border-radius:5px;padding:2px 7px;text-transform:none;letter-spacing:0}
.tag.warn{color:#ffc061;background:#1b160f;border-color:#7a5a1e}
.tag.bad{color:#ff8b8b;background:#1b0f0f;border-color:#7a2626}
.rep{font-weight:400;font-size:11px;color:#7d8b99;text-transform:none;letter-spacing:0}
</style>
</head>
<body>
<div class="wrap">
<h1>homelab &mdash; internal</h1>
<div class="muted">{{.Checked}} &bull; auto-refreshes every 60s &bull; TLS cert <span class="cert {{.CertClass}}">{{.CertDays}}</span></div>

<div class="rollup {{.StatusClass}}">{{.StatusText}}
{{if or .Criticals .Warnings}}<ul>{{range .Criticals}}<li class="bad">{{.}}</li>{{end}}{{range .Warnings}}<li class="warn">{{.}}</li>{{end}}</ul>{{end}}
</div>

<div class="panel">
<h2><span class="dot {{if .PiUp}}up{{else}}down{{end}}"></span> Pi-hole{{if .Blocking}} <span class="tag {{.BlockingClass}}">blocking {{.Blocking}}</span>{{end}}</h2>
<div class="grid">
<div class="stat"><div class="n">{{.Queries}}</div><div class="l">queries (24h)</div></div>
<div class="stat"><div class="n block">{{.Blocked}}</div><div class="l">blocked</div></div>
<div class="stat"><div class="n pct">{{.Pct}}</div><div class="l">blocked %</div></div>
<div class="stat"><div class="n">{{.Cache}}</div><div class="l">cache hit</div></div>
<div class="stat"><div class="n">{{.Blocklist}}</div><div class="l">blocklist</div></div>
<div class="stat"><div class="n">{{.Clients}}</div><div class="l">clients</div></div>
</div>
<div class="sparklbl" style="margin-top:12px">top blocked</div>
<table><tbody>
{{range .Top}}<tr><td class="c">{{.Count}}</td><td class="d">{{.Domain}}</td></tr>{{end}}
</tbody></table>
{{if .TopClients}}<div class="sparklbl" style="margin-top:12px">top clients</div>
<table><tbody>
{{range .TopClients}}<tr><td class="c">{{.Count}}</td><td class="d">{{.Domain}}</td></tr>{{end}}
</tbody></table>{{end}}
{{if .PiSpark}}<div class="sparkbox" style="margin-top:14px"><div class="sparklbl">blocked % &middot; 24h</div>{{.PiSpark}}</div>{{end}}
</div>

<div class="panel">
<h2><span class="dot {{if .MCUp}}up{{else}}down{{end}}"></span> Minecraft</h2>
<div class="grid">
<div class="stat"><div class="n">{{.Online}}</div><div class="l">online</div></div>
<div class="stat"><div class="n">{{.TPS}}</div><div class="l">TPS</div></div>
<div class="stat"><div class="n">{{.Day}}</div><div class="l">day</div></div>
</div>
</div>

{{range .Hosts}}
<div class="panel">
<h2><span class="dot {{if .Up}}up{{else}}down{{end}}"></span> {{.Box}} <span class="rep">reported {{.Reported}}</span>{{if .Stale}} <span class="tag bad">stale</span>{{end}}{{if .CIBuilding}} <span class="tag warn">building</span>{{end}}{{if .RebootReq}} <span class="tag bad">reboot</span>{{end}}</h2>
<div class="grid">
<div class="stat"><div class="n" style="font-size:22px">{{.Uptime}}</div><div class="l">uptime</div></div>
<div class="stat"><div class="n {{.LoadClass}}" style="font-size:22px">{{.Load}}</div><div class="l">load 1m</div></div>
<div class="stat"><div class="n {{.MemClass}}" style="font-size:22px">{{.Mem}}</div><div class="l">memory</div></div>
<div class="stat"><div class="n {{.SwapClass}}" style="font-size:22px">{{.Swap}}</div><div class="l">swap</div></div>
<div class="stat"><div class="n {{.TempClass}}" style="font-size:22px">{{.Temp}}</div><div class="l">temp</div></div>
{{if .FailedUnits}}<div class="stat"><div class="n {{.FailedClass}}" style="font-size:22px">{{.FailedUnits}}</div><div class="l">failed units</div></div>{{end}}
{{if .Updates}}<div class="stat"><div class="n" style="font-size:22px">{{.Updates}}</div><div class="l">updates</div></div>{{end}}
{{if .Backup}}<div class="stat"><div class="n {{.BackupClass}}" style="font-size:16px">{{.Backup}}</div><div class="l">mc backup</div></div>{{end}}
</div>
{{range .Disks}}<div class="disk"><div class="dlabel">{{.Mount}} <span class="muted">{{.Used}}</span></div><div class="bar"><div class="fill {{.BarClass}}" style="width:{{.Pct}}%"></div></div></div>{{end}}
{{if or .LoadSpark .DiskSpark}}<div class="sparks">{{if .LoadSpark}}<div class="sparkbox"><div class="sparklbl">load &middot; 24h</div>{{.LoadSpark}}</div>{{end}}{{if .DiskSpark}}<div class="sparkbox"><div class="sparklbl">disk % &middot; 24h</div>{{.DiskSpark}}</div>{{end}}</div>{{end}}
{{if .Services}}<div class="svcs">{{range .Services}}<span class="svc"><span class="dot sm {{if .Up}}up{{else}}down{{end}}"></span>{{.Name}}</span>{{end}}</div>{{end}}
</div>
{{end}}
</div>
</body>
</html>`
