// A compact "now playing" bar shown on every page except the full tunes page.
// It restores the track + position from shared state and follows you across
// navigations (resuming where you left off; auto-play if the browser allows it,
// otherwise one click). It self-mounts: no HTML or CSS link needed per page.

import { loadState, RepeatMode, saveState } from "./playerstate.js";

interface Tune {
    file: string;
    name: string;
    size: number;
}

function fileURL(file: string): string {
    return `/tunes/file/${encodeURIComponent(file)}`;
}

function clamp(v: number, min: number, max: number): number {
    return Math.max(min, Math.min(max, v));
}

function injectCSS() {
    if (document.getElementById("miniPlayerCss") !== null) {
        return;
    }
    const link = document.createElement("link");
    link.id = "miniPlayerCss";
    link.rel = "stylesheet";
    link.href = "/static/css/miniplayer.css";
    document.head.appendChild(link);
}

function button(label: string, title: string): HTMLButtonElement {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "miniBtn";
    b.textContent = label;
    b.title = title;
    return b;
}

export async function initMiniPlayer() {
    const saved = loadState();
    if (saved === null) {
        return; // nothing has been played yet, so no bar
    }

    let tunes: Tune[] = [];
    try {
        const resp = await fetch("/tunes/list");
        tunes = (await resp.json()) as Tune[];
    } catch {
        tunes = [];
    }
    if (tunes.length === 0) {
        return;
    }

    let idx = tunes.findIndex((t) => t.file === saved.file);
    if (idx < 0) {
        idx = 0;
    }
    const shuffle = saved.shuffle;
    const repeat: RepeatMode = saved.repeat;

    const audio = new Audio();
    audio.volume = saved.vol / 100;

    injectCSS();

    const bar = document.createElement("div");
    bar.id = "miniPlayer";
    const prev = button("◀◀", "previous");
    const play = button("▶", "play / pause");
    const next = button("▶▶", "next");
    const name = document.createElement("span");
    name.className = "miniName";
    const seek = document.createElement("div");
    seek.className = "miniSeek";
    const fill = document.createElement("div");
    fill.className = "miniFill";
    seek.appendChild(fill);
    const link = document.createElement("a");
    link.className = "miniLink";
    link.href = "/tunes/home";
    link.textContent = "tunes";
    link.title = "open the full player";
    const close = button("x", "close");
    close.classList.add("miniClose");
    bar.append(prev, play, next, name, seek, link, close);
    document.body.appendChild(bar);

    function pick(auto: boolean): number {
        if (auto && repeat === "one") {
            return idx;
        }
        if (shuffle && tunes.length > 1) {
            let r = idx;
            while (r === idx) {
                r = Math.floor(Math.random() * tunes.length);
            }
            return r;
        }
        if (idx >= tunes.length - 1) {
            return auto && repeat === "off" ? -1 : 0;
        }
        return idx + 1;
    }

    function setTrack(i: number, playNow: boolean) {
        idx = i;
        audio.src = fileURL(tunes[i].file);
        name.textContent = tunes[i].name;
        saveState({ file: tunes[i].file });
        if (playNow) {
            void audio.play().catch(() => undefined);
        }
    }

    // Initial restore: load the saved track and seek to the saved position once
    // metadata is available, then try to resume if it was playing.
    let restoreTime = saved.time;
    audio.src = fileURL(tunes[idx].file);
    name.textContent = tunes[idx].name;
    audio.addEventListener("loadedmetadata", () => {
        if (restoreTime > 0 && isFinite(audio.duration)) {
            audio.currentTime = Math.min(restoreTime, audio.duration - 0.1);
            restoreTime = 0;
        }
    }, { once: true });
    if (saved.playing) {
        void audio.play().catch(() => undefined); // may be blocked until a click
    }

    prev.addEventListener("click", () => setTrack(idx <= 0 ? tunes.length - 1 : idx - 1, true));
    next.addEventListener("click", () => {
        const i = pick(false);
        if (i >= 0) {
            setTrack(i, true);
        }
    });
    play.addEventListener("click", () => {
        if (audio.paused) {
            void audio.play().catch(() => undefined);
        } else {
            audio.pause();
        }
    });
    close.addEventListener("click", () => {
        audio.pause();
        saveState({ file: "", playing: false });
        bar.remove();
    });
    seek.addEventListener("click", (e) => {
        const rect = seek.getBoundingClientRect();
        if (isFinite(audio.duration)) {
            audio.currentTime = ((e.clientX - rect.left) / rect.width) * audio.duration;
        }
    });

    audio.addEventListener("play", () => {
        play.textContent = "||";
        saveState({ playing: true });
    });
    audio.addEventListener("pause", () => {
        play.textContent = "▶";
        saveState({ playing: false });
    });
    audio.addEventListener("ended", () => {
        const i = pick(true);
        if (i >= 0) {
            setTrack(i, true);
        }
    });
    let lastSaved = -2;
    audio.addEventListener("timeupdate", () => {
        const pct = audio.duration ? (audio.currentTime / audio.duration) * 100 : 0;
        fill.style.width = `${pct}%`;
        if (Math.abs(audio.currentTime - lastSaved) >= 1) {
            lastSaved = audio.currentTime;
            saveState({ time: audio.currentTime });
        }
    });
    // Save the exact position right before the page unloads, so the next page
    // resumes precisely.
    window.addEventListener("beforeunload", () => saveState({ time: audio.currentTime }));

    // Drag the bar by its body (not the controls); remember where it sits.
    try {
        const x = localStorage.getItem("tunes.mini.x");
        const y = localStorage.getItem("tunes.mini.y");
        if (x !== null && y !== null) {
            bar.style.left = `${clamp(Number(x), 0, window.innerWidth - 80)}px`;
            bar.style.top = `${clamp(Number(y), 0, window.innerHeight - 30)}px`;
            bar.style.right = "auto";
            bar.style.bottom = "auto";
        }
    } catch {
        // ignore
    }
    let dragging = false;
    let offX = 0;
    let offY = 0;
    bar.addEventListener("pointerdown", (e) => {
        const target = e.target as Element | null;
        if (target && target.closest("button, a, .miniSeek")) {
            return;
        }
        dragging = true;
        const rect = bar.getBoundingClientRect();
        offX = e.clientX - rect.left;
        offY = e.clientY - rect.top;
        bar.style.right = "auto";
        bar.style.bottom = "auto";
        bar.setPointerCapture(e.pointerId);
        e.preventDefault();
    });
    bar.addEventListener("pointermove", (e) => {
        if (!dragging) {
            return;
        }
        bar.style.left = `${clamp(e.clientX - offX, 0, window.innerWidth - bar.offsetWidth)}px`;
        bar.style.top = `${clamp(e.clientY - offY, 0, window.innerHeight - 30)}px`;
    });
    const endDrag = () => {
        if (!dragging) {
            return;
        }
        dragging = false;
        try {
            localStorage.setItem("tunes.mini.x", String(parseInt(bar.style.left, 10) || 0));
            localStorage.setItem("tunes.mini.y", String(parseInt(bar.style.top, 10) || 0));
        } catch {
            // ignore
        }
    };
    bar.addEventListener("pointerup", endDrag);
    bar.addEventListener("pointercancel", endDrag);
}
