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
