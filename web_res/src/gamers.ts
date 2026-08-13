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

interface Valheim {
  up?: boolean;
  health?: string;
  uptimeSec?: number;
  players?: string[];
  playerCount?: number;
  world?: string;
  seed?: string;
  worldSizeBytes?: number;
  version?: string;
  mods?: string[];
  backupAgeH?: number;
  backupCount?: number;
  stale?: boolean;
  known?: boolean;
}

// dash writes a value into a dashboard cell, or a placeholder when the value is
// missing (nothing pushed yet).
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

// fmtSize renders the world .db size. Worth showing because a Valheim world
// grows steadily as it is explored, so it doubles as a "how much have we done".
function fmtSize(bytes: number): string {
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + "G";
  if (bytes >= 1048576) return Math.round(bytes / 1048576) + "M";
  return Math.round(bytes / 1024) + "K";
}

// backupClass colours backup freshness. The job runs daily at 04:00, so a gap
// past ~26h means it missed a run, and past 48h it has missed two.
function backupClass(ageH: number): string {
  if (ageH >= 48) return "bad";
  if (ageH >= 26) return "warn";
  return "good";
}

// Live server status. Valheim gives no player count over the wire (see
// valheim.go), so the names come from the server log and there are no avatars to
// render, unlike the Minecraft version this replaced.
function refreshStatus(): void {
  fetch("/gamers/valheim")
    .then((r) => r.json())
    .then((d: Valheim) => {
      const el = document.getElementById("vhstatus");
      const names = document.getElementById("vhplayers");

      // known=false means the collector has never reported. That is a different
      // thing from an empty server, and saying so beats implying nobody is on.
      if (el) {
        if (!d.known) el.textContent = "STATUS UNKNOWN";
        else if (!d.up) el.textContent = "SERVER OFFLINE";
        else {
          const n = d.playerCount ?? 0;
          el.textContent = n === 1 ? "1 VIKING ONLINE" : `${n} VIKINGS ONLINE`;
        }
      }

      if (names) {
        names.textContent = "";
        for (const p of d.players ?? []) {
          const fig = document.createElement("span");
          fig.className = "viking";
          // textContent, never innerHTML: a player picks their own character
          // name, so this string is outside our control.
          fig.textContent = p;
          names.appendChild(fig);
        }
      }

      const suffix = d.stale ? " (stale)" : "";
      dash("dPlayers", d.known && typeof d.playerCount === "number" ? String(d.playerCount) + suffix : null);
      dash("dUptime", d.known && typeof d.uptimeSec === "number" ? fmtUptime(d.uptimeSec) : null);
      dash("dVersion", d.known && d.version ? d.version : null);
      dash("dWorldSize", d.known && typeof d.worldSizeBytes === "number" ? fmtSize(d.worldSizeBytes) : null);

      const bk = document.getElementById("dBackup");
      if (bk) {
        if (d.known && typeof d.backupAgeH === "number" && d.backupCount) {
          bk.textContent = d.backupAgeH.toFixed(1) + "h";
          bk.className = "val " + backupClass(d.backupAgeH);
        } else {
          bk.textContent = "--";
          bk.className = "val";
        }
      }

      const seedEl = document.getElementById("vhseed");
      if (seedEl && d.seed) seedEl.textContent = d.seed;
      const worldEl = document.getElementById("vhworld");
      if (worldEl && d.world) worldEl.textContent = d.world;
      const modsEl = document.getElementById("vhmods");
      if (modsEl) modsEl.textContent = (d.mods ?? []).join(" \u2022 ") || "vanilla";
    })
    .catch(() => {
      /* leave the placeholders on error */
    });
}
refreshStatus();
setInterval(refreshStatus, 20000);
