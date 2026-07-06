// Fetches the real visit count from the server and renders it in the old-school
// LCD counter, zero-padded with thin spaces between digits.
fetch("/gamers/count")
  .then((r) => r.json())
  .then((d: { visits?: number }) => {
    const el = document.getElementById("visits");
    if (!el) return;
    let n = String(d.visits ?? 0);
    while (n.length < 6) n = "0" + n;
    el.textContent = n.split("").join(" ");
  })
  .catch(() => {
    /* leave the placeholder if the count can't be fetched */
  });

// Live Minecraft server status: online count + player names, polled periodically.
function refreshStatus(): void {
  fetch("/gamers/status")
    .then((r) => r.json())
    .then((d: { up?: boolean; online?: number; max?: number; players?: string[] }) => {
      const el = document.getElementById("mcstatus");
      if (!el) return;
      if (!d.up) {
        el.textContent = "SERVER OFFLINE";
        return;
      }
      const who = d.players && d.players.length ? " — " + d.players.join(", ") : "";
      el.textContent = `${d.online ?? 0}/${d.max ?? 0} PLAYING${who}`;
    })
    .catch(() => {
      /* leave the placeholder on error */
    });
}
refreshStatus();
setInterval(refreshStatus, 20000);
