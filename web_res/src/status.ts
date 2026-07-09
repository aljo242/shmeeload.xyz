// Status page: reads the merged /gamers/live feed and paints a colored dot per
// service. The site itself is trivially "up" (it served this page); everything
// else is derived from the feed and the game host's pushed service map.
type Dot = "up" | "warn" | "down" | "unknown";

interface MCPlayer {
  name: string;
  uuid: string;
}

interface Live {
  up?: boolean;
  online?: number;
  players?: MCPlayer[];
  tps?: number;
  uptimeSec?: number;
  services?: Record<string, string>;
  stale?: boolean;
  pushedAt?: number;
}

// renderHeads paints the online players as head avatars (same same-origin proxy
// the /gamers page uses), or a placeholder line when nobody is on.
function renderHeads(d: Live): void {
  const count = document.getElementById("online-count");
  if (count) count.textContent = d.up ? String(d.online ?? 0) : "--";
  const box = document.getElementById("mcheads");
  if (!box) return;
  box.textContent = "";
  const players = d.up ? d.players ?? [] : [];
  if (players.length === 0) {
    const none = document.createElement("div");
    none.className = "none";
    none.textContent = d.up ? "nobody on right now" : "server offline";
    box.appendChild(none);
    return;
  }
  for (const p of players) {
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
    box.appendChild(fig);
  }
}

function setRow(idBase: string, dot: Dot, detail: string): void {
  const d = document.getElementById(idBase + "-dot");
  const t = document.getElementById(idBase + "-detail");
  if (d) d.className = "dot " + (dot === "unknown" ? "" : dot);
  if (t) t.textContent = detail;
}

// classify maps a free-form service word from the push into a dot color.
function classify(v: string): Dot {
  const s = v.toLowerCase();
  if (["up", "ok", "idle", "online", "active", "running", "healthy"].includes(s)) return "up";
  if (["busy", "warn", "degraded", "slow", "stale"].includes(s)) return "warn";
  if (["down", "fail", "failed", "offline", "error", "dead"].includes(s)) return "down";
  return "unknown";
}

function titleize(k: string): string {
  return k.charAt(0).toUpperCase() + k.slice(1).replace(/[_-]+/g, " ");
}

function refresh(): void {
  fetch("/gamers/live")
    .then((r) => r.json())
    .then((d: Live) => {
      renderHeads(d);

      // Minecraft server: down if SLP is offline; degraded if slow or stale; else up.
      if (!d.up) {
        setRow("mc", "down", "offline");
      } else if (typeof d.tps === "number" && (d.tps < 15 || d.stale)) {
        setRow("mc", "warn", `${d.tps.toFixed(1)} TPS`);
      } else {
        setRow("mc", "up", typeof d.tps === "number" ? `${d.tps.toFixed(1)} TPS` : "online");
      }

      // Live feed: has the game host pushed recently?
      if (!d.pushedAt) {
        setRow("feed", "down", "no data");
      } else if (d.stale) {
        setRow("feed", "warn", "stale");
      } else {
        setRow("feed", "up", "fresh");
      }

      // Extra services reported by the push (foundry, CI, Pi-hole, ...).
      const box = document.getElementById("services");
      if (box) {
        box.textContent = "";
        for (const [k, v] of Object.entries(d.services ?? {})) {
          const row = document.createElement("div");
          row.className = "row";
          const dot = document.createElement("span");
          dot.className = "dot " + (classify(v) === "unknown" ? "" : classify(v));
          const label = document.createElement("span");
          label.className = "rlabel";
          label.textContent = titleize(k);
          const detail = document.createElement("span");
          detail.className = "rdetail";
          detail.textContent = v;
          row.appendChild(dot);
          row.appendChild(label);
          row.appendChild(detail);
          box.appendChild(row);
        }
      }
    })
    .catch(() => {
      setRow("mc", "unknown", "unreachable");
      setRow("feed", "unknown", "unreachable");
    });
}
refresh();
setInterval(refresh, 20000);
