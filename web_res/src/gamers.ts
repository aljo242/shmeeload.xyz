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

// Live Minecraft server status: online count in the banner, plus a row of
// player-head avatars (served same-origin from /gamers/head so the CSP holds).
function refreshStatus(): void {
  fetch("/gamers/status")
    .then((r) => r.json())
    .then((d: { up?: boolean; online?: number; max?: number; players?: MCPlayer[] }) => {
      const el = document.getElementById("mcstatus");
      const heads = document.getElementById("mcheads");
      if (heads) heads.textContent = "";
      if (!el) return;
      if (!d.up) {
        el.textContent = "SERVER OFFLINE";
        return;
      }
      el.textContent = `${d.online ?? 0}/${d.max ?? 0} PLAYING`;
      if (!heads) return;
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
    })
    .catch(() => {
      /* leave the placeholder on error */
    });
}
refreshStatus();
setInterval(refreshStatus, 20000);
