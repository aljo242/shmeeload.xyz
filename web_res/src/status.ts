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
    })
    .catch(() => {
      setRow("mc", "unknown", "unreachable");
      setRow("feed", "unknown", "unreachable");
    });
}
refresh();
setInterval(refresh, 20000);
