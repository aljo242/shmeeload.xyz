// Fetches the real visit count from the server and renders it in the old-school
// mechanical odometer, zero-padded, one digit per cell.
fetch("/gamers/count")
  .then((r) => r.json())
  .then((d: { visits?: number }) => {
    const el = document.getElementById("visits");
    if (!el) return;
    let n = String(d.visits ?? 0);
    while (n.length < 6) n = "0" + n;
    el.textContent = "";
    for (const ch of n) {
      const cell = document.createElement("span");
      cell.className = "digit";
      cell.textContent = ch;
      el.appendChild(cell);
    }
  })
  .catch(() => {
    /* leave the placeholder if the count can't be fetched */
  });

interface MCPlayer {
  name: string;
  uuid: string;
}

interface Live {
  up?: boolean;
  online?: number;
  max?: number;
  players?: MCPlayer[];
  tps?: number;
  msPerTick?: number;
  day?: number;
  uptimeSec?: number;
  whitelist?: number;
  stale?: boolean;
}

// dash writes a value into a dashboard cell, or an em-free placeholder when the
// value is missing (nothing pushed yet).
function dash(id: string, value: string | null): void {
  const el = document.getElementById(id);
  if (el) el.textContent = value ?? "--";
}

// fmtUptime turns a second count into a compact "3d 4h" / "4h 12m" / "12m".
function fmtUptime(sec: number): string {
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

// tpsClass buckets tick rate into a color band for the dashboard.
function tpsClass(tps: number): string {
  if (tps >= 19) return "good";
  if (tps >= 12) return "warn";
  return "bad";
}

// Live server status: online count + heads in the banner, plus the dashboard
// (TPS, in-game day, uptime, whitelist) from the pushed telemetry.
function refreshStatus(): void {
  fetch("/gamers/live")
    .then((r) => r.json())
    .then((d: Live) => {
      const el = document.getElementById("mcstatus");
      const heads = document.getElementById("mcheads");
      if (heads) heads.textContent = "";
      if (el) el.textContent = d.up ? `${d.online ?? 0}/${d.max ?? 0} PLAYING` : "SERVER OFFLINE";

      if (heads && d.up) {
        for (const p of d.players ?? []) {
          const fig = document.createElement("span");
          fig.className = "head";
          const img = document.createElement("img");
          img.alt = p.name;
          img.width = 40;
          img.height = 40;
          img.src = "/gamers/head/" + encodeURIComponent(p.uuid || p.name);
          img.onerror = (): void => img.remove();
          const label = document.createElement("span");
          label.className = "hname";
          label.textContent = p.name;
          fig.appendChild(img);
          fig.appendChild(label);
          heads.appendChild(fig);
        }
      }

      // Dashboard cells. A pushed field may be absent (never pushed) or stale.
      const tpsEl = document.getElementById("dTps");
      if (tpsEl) {
        if (typeof d.tps === "number") {
          tpsEl.textContent = d.tps.toFixed(1) + (d.stale ? " (stale)" : "");
          tpsEl.className = "val " + (d.stale ? "warn" : tpsClass(d.tps));
        } else {
          tpsEl.textContent = "--";
          tpsEl.className = "val";
        }
      }
      dash("dDay", typeof d.day === "number" ? String(d.day) : null);
      dash("dUptime", typeof d.uptimeSec === "number" ? fmtUptime(d.uptimeSec) : null);
      dash("dWhitelist", typeof d.whitelist === "number" ? String(d.whitelist) : null);
    })
    .catch(() => {
      /* leave the placeholders on error */
    });
}
refreshStatus();
setInterval(refreshStatus, 20000);
