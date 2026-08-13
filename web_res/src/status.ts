// Status page: reads the merged /gamers/live feed and paints a colored dot per
// service. The site itself is trivially "up" (it served this page); everything
// else is derived from the feed and the game host's pushed service map.
type Dot = "up" | "warn" | "down" | "unknown";

interface Valheim {
  up?: boolean;
  players?: string[];
  playerCount?: number;
  world?: string;
  version?: string;
  stale?: boolean;
  known?: boolean;
}

// renderPlayers lists who is on. Valheim has no avatar API, so these are name
// chips rather than the head images the Minecraft version proxied.
function renderPlayers(d: Valheim): void {
  const count = document.getElementById("online-count");
  if (count) count.textContent = d.known && d.up ? String(d.playerCount ?? 0) : "--";
  const box = document.getElementById("vhplayers");
  if (!box) return;
  box.textContent = "";
  const players = d.up ? d.players ?? [] : [];
  if (players.length === 0) {
    const none = document.createElement("div");
    none.className = "none";
    none.textContent = !d.known ? "status unknown" : d.up ? "nobody on right now" : "server offline";
    box.appendChild(none);
    return;
  }
  for (const p of players) {
    const fig = document.createElement("span");
    fig.className = "hname";
    // textContent, not innerHTML: a player names their own character.
    fig.textContent = p;
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
  fetch("/gamers/valheim")
    .then((r) => r.json())
    .then((d: Valheim) => {
      renderPlayers(d);

      // Never pushed is its own state: a dead collector is not an idle server.
      if (!d.known) {
        setRow("game", "unknown", "no data");
      } else if (!d.up) {
        setRow("game", "down", "offline");
      } else if (d.stale) {
        setRow("game", "warn", "stale");
      } else {
        setRow("game", "up", d.version ? "v" + d.version : "online");
      }

      // Is the collector still reporting?
      if (!d.known) {
        setRow("feed", "down", "no data");
      } else if (d.stale) {
        setRow("feed", "warn", "stale");
      } else {
        setRow("feed", "up", "fresh");
      }
    })
    .catch(() => {
      setRow("game", "unknown", "unreachable");
      setRow("feed", "unknown", "unreachable");
    });
}
refresh();
setInterval(refresh, 20000);
