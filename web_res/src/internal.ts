// Internal homelab view: renders the Pi-hole stats and a Minecraft summary from
// the gated /internal/live endpoint. This page sits behind Basic Auth; none of
// this data is exposed on a public route.
export {}; // module scope so top-level names don't collide with other page scripts

interface DomainCount {
  domain: string;
  count: number;
}

interface Pihole {
  up?: boolean;
  queriesToday?: number;
  blockedToday?: number;
  percentBlocked?: number;
  blocklistSize?: number;
  activeClients?: number;
  topBlocked?: DomainCount[];
  checkedAt?: number;
}

interface MC {
  up?: boolean;
  online?: number;
  max?: number;
  tps?: number;
  day?: number;
}

interface Internal {
  pihole?: Pihole;
  mc?: MC;
}

function num(n?: number): string {
  return typeof n === "number" ? n.toLocaleString() : "--";
}

function set(id: string, v: string): void {
  const el = document.getElementById(id);
  if (el) el.textContent = v;
}

function dot(id: string, up?: boolean): void {
  const el = document.getElementById(id);
  if (el) el.className = "dot " + (up ? "up" : "down");
}

function refresh(): void {
  fetch("/internal/live")
    .then((r) => r.json())
    .then((d: Internal) => {
      const p = d.pihole ?? {};
      dot("ph-dot", p.up);
      set("ph-queries", num(p.queriesToday));
      set("ph-blocked", num(p.blockedToday));
      set("ph-pct", typeof p.percentBlocked === "number" ? p.percentBlocked.toFixed(1) + "%" : "--");
      set("ph-list", num(p.blocklistSize));
      set("ph-clients", num(p.activeClients));

      const top = document.getElementById("ph-top");
      if (top) {
        top.textContent = "";
        for (const x of p.topBlocked ?? []) {
          const tr = document.createElement("tr");
          const c = document.createElement("td");
          c.className = "c";
          c.textContent = x.count.toLocaleString();
          const dd = document.createElement("td");
          dd.className = "d";
          dd.textContent = x.domain;
          tr.appendChild(c);
          tr.appendChild(dd);
          top.appendChild(tr);
        }
      }

      const mc = d.mc ?? {};
      dot("mc-dot", mc.up);
      set("mc-online", mc.up ? `${mc.online ?? 0}/${mc.max ?? 0}` : "--");
      set("mc-tps", typeof mc.tps === "number" ? mc.tps.toFixed(1) : "--");
      set("mc-day", num(mc.day));

      const chk = document.getElementById("checked");
      if (chk && p.checkedAt) chk.textContent = "pi-hole updated " + new Date(p.checkedAt * 1000).toLocaleTimeString();
    })
    .catch(() => {
      /* keep last values on a transient error */
    });
}
refresh();
setInterval(refresh, 30000);
