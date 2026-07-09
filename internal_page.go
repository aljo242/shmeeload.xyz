package main

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// The internal homelab page is rendered server-side and refreshed by a plain
// <meta refresh>. Page navigations carry Basic Auth reliably (unlike fetch), so
// this needs no cookie, token, or client script.

type internalRow struct{ Count, Domain string }

type internalView struct {
	PiUp                                      bool
	Queries, Blocked, Pct, Blocklist, Clients string
	Top                                       []internalRow
	MCUp                                      bool
	Online, TPS, Day                          string
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

func buildInternalView(pi piholeStats, mc liveResponse) internalView {
	v := internalView{PiUp: pi.Up, MCUp: mc.Up}

	if pi.Up {
		v.Queries = commafy(pi.QueriesToday)
		v.Blocked = commafy(pi.BlockedToday)
		v.Pct = strconv.FormatFloat(pi.PercentBlocked, 'f', 1, 64) + "%"
		v.Blocklist = commafy(pi.BlocklistSize)
		v.Clients = commafy(pi.ActiveClients)
		for _, d := range pi.TopBlocked {
			v.Top = append(v.Top, internalRow{Count: commafy(d.Count), Domain: d.Domain})
		}
	} else {
		v.Queries, v.Blocked, v.Pct, v.Blocklist, v.Clients = "--", "--", "--", "--", "--"
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
.panel{background:#121821;border:1px solid #223041;border-radius:10px;padding:16px 18px;margin:16px 0}
.panel h2{font-size:14px;text-transform:uppercase;letter-spacing:1px;color:#8fb7ff;margin:0 0 12px;display:flex;align-items:center;gap:8px}
.dot{width:10px;height:10px;border-radius:50%;background:#556;display:inline-block}
.dot.up{background:#37d67a;box-shadow:0 0 8px #37d67a66}
.dot.down{background:#ff5a5a;box-shadow:0 0 8px #ff5a5a66}
.grid{display:flex;flex-wrap:wrap;gap:22px}
.stat .n{font-size:30px;font-weight:800;line-height:1.1}
.stat .l{font-size:12px;color:#7d8b99;text-transform:uppercase;letter-spacing:.5px}
.n.block{color:#ff9d3c}.n.pct{color:#37d67a}
table{width:100%;border-collapse:collapse;margin-top:6px;font-size:14px}
td{padding:6px 4px;border-bottom:1px solid #1b2431}
td.c{text-align:right;color:#ff9d3c;font-variant-numeric:tabular-nums;width:72px;font-weight:700}
td.d{color:#cfe;word-break:break-all;font-family:ui-monospace,"SF Mono",Menlo,monospace;font-size:12.5px}
</style>
</head>
<body>
<div class="wrap">
<h1>homelab &mdash; internal</h1>
<div class="muted">{{.Checked}} &bull; auto-refreshes every 60s</div>

<div class="panel">
<h2><span class="dot {{if .PiUp}}up{{else}}down{{end}}"></span> Pi-hole</h2>
<div class="grid">
<div class="stat"><div class="n">{{.Queries}}</div><div class="l">queries (24h)</div></div>
<div class="stat"><div class="n block">{{.Blocked}}</div><div class="l">blocked</div></div>
<div class="stat"><div class="n pct">{{.Pct}}</div><div class="l">blocked %</div></div>
<div class="stat"><div class="n">{{.Blocklist}}</div><div class="l">blocklist</div></div>
<div class="stat"><div class="n">{{.Clients}}</div><div class="l">clients</div></div>
</div>
<table><tbody>
{{range .Top}}<tr><td class="c">{{.Count}}</td><td class="d">{{.Domain}}</td></tr>{{end}}
</tbody></table>
</div>

<div class="panel">
<h2><span class="dot {{if .MCUp}}up{{else}}down{{end}}"></span> Minecraft</h2>
<div class="grid">
<div class="stat"><div class="n">{{.Online}}</div><div class="l">online</div></div>
<div class="stat"><div class="n">{{.TPS}}</div><div class="l">TPS</div></div>
<div class="stat"><div class="n">{{.Day}}</div><div class="l">day</div></div>
</div>
</div>
</div>
</body>
</html>`
