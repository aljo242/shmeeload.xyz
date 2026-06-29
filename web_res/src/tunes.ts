import { getElement } from "./dom.js";
import { initTheme } from "./theme.js";

interface Tune {
    file: string;
    name: string;
    size: number;
}

type RepeatMode = "off" | "all" | "one";

let tunes: Tune[] = [];
let current = -1;
let shuffle = false;
let repeat: RepeatMode = "off";
const audio = new Audio();
audio.preload = "none";

function fmtTime(seconds: number): string {
    if (!isFinite(seconds) || seconds < 0) {
        return "0:00";
    }
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, "0")}`;
}

function clamp(v: number, min: number, max: number): number {
    return Math.max(min, Math.min(max, v));
}

function fileURL(t: Tune): string {
    return `/tunes/file/${encodeURIComponent(t.file)}`;
}

function playlistItems(): NodeListOf<HTMLLIElement> {
    return document.querySelectorAll<HTMLLIElement>("#playlist li");
}

function highlight() {
    playlistItems().forEach((li, i) => li.classList.toggle("playing", i === current));
    const el = playlistItems()[current];
    if (el !== undefined) {
        el.scrollIntoView({ block: "nearest" });
    }
}

function load(i: number, play = true) {
    if (i < 0 || i >= tunes.length) {
        return;
    }
    current = i;
    audio.src = fileURL(tunes[i]);
    getElement("trackName").textContent = tunes[i].name;
    highlight();
    if (play) {
        void audio.play().catch(() => undefined);
    }
}

// pickIndex chooses the next track, honoring shuffle and repeat. auto is true
// when the current track ended on its own (vs the user pressing next): only then
// does "repeat one" replay and "repeat off" stop (-1) at the end of the list.
function pickIndex(auto: boolean): number {
    if (tunes.length === 0) {
        return -1;
    }
    if (auto && repeat === "one") {
        return current;
    }
    if (shuffle) {
        if (tunes.length === 1) {
            return current;
        }
        let r = current;
        while (r === current) {
            r = Math.floor(Math.random() * tunes.length);
        }
        return r;
    }
    if (current >= tunes.length - 1) {
        return auto && repeat === "off" ? -1 : 0;
    }
    return current + 1;
}

function next() {
    const i = pickIndex(false);
    if (i >= 0) {
        load(i);
    }
}

function prev() {
    load(current <= 0 ? tunes.length - 1 : current - 1);
}

function onEnded() {
    const i = pickIndex(true);
    if (i >= 0) {
        load(i);
    }
}

function updateToggles() {
    const sh = getElement("shuffle");
    sh.classList.toggle("active", shuffle);
    sh.setAttribute("aria-pressed", shuffle ? "true" : "false");
    const rp = getElement("repeat");
    rp.classList.toggle("active", repeat !== "off");
    rp.setAttribute("aria-pressed", repeat !== "off" ? "true" : "false");
    rp.textContent = repeat === "one" ? "rpt 1" : "rpt";
    rp.title = `repeat: ${repeat}`;
}

function loadPrefs() {
    try {
        shuffle = localStorage.getItem("tunes.shuffle") === "1";
        const r = localStorage.getItem("tunes.repeat");
        if (r === "all" || r === "one") {
            repeat = r;
        }
    } catch {
        // localStorage may be unavailable (private mode); defaults are fine.
    }
}

function savePref(key: string, value: string) {
    try {
        localStorage.setItem(key, value);
    } catch {
        // ignore
    }
}

function buildPlaylist() {
    const ol = getElement("playlist");
    ol.textContent = "";
    if (tunes.length === 0) {
        const li = document.createElement("li");
        li.className = "empty";
        li.textContent = "no tracks yet";
        ol.appendChild(li);
        return;
    }
    tunes.forEach((t, i) => {
        const li = document.createElement("li");
        const num = document.createElement("span");
        num.className = "num";
        num.textContent = `${i + 1}.`;
        const title = document.createElement("span");
        title.className = "ptitle";
        title.textContent = t.name;
        const dur = document.createElement("span");
        dur.className = "pdur";
        dur.textContent = "-:--";
        li.append(num, title, dur);
        li.addEventListener("click", () => load(i));
        ol.appendChild(li);

        // Read each track's duration lazily (a metadata-only request).
        const probe = new Audio();
        probe.preload = "metadata";
        probe.src = fileURL(t);
        probe.addEventListener("loadedmetadata", () => {
            dur.textContent = fmtTime(probe.duration);
        });
    });
}

async function loadList() {
    try {
        const resp = await fetch("/tunes/list");
        tunes = (await resp.json()) as Tune[];
    } catch {
        tunes = [];
    }
    buildPlaylist();
}

// makeDraggable lets the player be moved by its title bar, persisting the spot.
function makeDraggable() {
    const player = getElement("player");
    const handle = getElement("titlebar");
    let dragging = false;
    let offX = 0;
    let offY = 0;

    try {
        const x = localStorage.getItem("tunes.x");
        const y = localStorage.getItem("tunes.y");
        if (x !== null && y !== null) {
            player.style.left = `${clamp(Number(x), 0, window.innerWidth - 80)}px`;
            player.style.top = `${clamp(Number(y), 0, window.innerHeight - 40)}px`;
        }
    } catch {
        // ignore
    }

    handle.addEventListener("pointerdown", (e) => {
        dragging = true;
        const rect = player.getBoundingClientRect();
        offX = e.clientX - rect.left;
        offY = e.clientY - rect.top;
        handle.setPointerCapture(e.pointerId);
        e.preventDefault();
    });
    handle.addEventListener("pointermove", (e) => {
        if (!dragging) {
            return;
        }
        player.style.left = `${clamp(e.clientX - offX, 0, window.innerWidth - player.offsetWidth)}px`;
        player.style.top = `${clamp(e.clientY - offY, 0, window.innerHeight - 40)}px`;
    });
    const end = () => {
        if (!dragging) {
            return;
        }
        dragging = false;
        savePref("tunes.x", String(parseInt(player.style.left, 10) || 0));
        savePref("tunes.y", String(parseInt(player.style.top, 10) || 0));
    };
    handle.addEventListener("pointerup", end);
    handle.addEventListener("pointercancel", end);
}

window.onload = async () => {
    initTheme();
    loadPrefs();
    updateToggles();
    makeDraggable();
    await loadList();

    getElement("play").addEventListener("click", () => {
        if (current === -1) {
            if (tunes.length > 0) {
                load(0);
            }
            return;
        }
        if (audio.paused) {
            void audio.play().catch(() => undefined);
        } else {
            audio.pause();
        }
    });
    getElement("stop").addEventListener("click", () => {
        audio.pause();
        audio.currentTime = 0;
    });
    getElement("prev").addEventListener("click", prev);
    getElement("next").addEventListener("click", next);
    getElement("shuffle").addEventListener("click", () => {
        shuffle = !shuffle;
        savePref("tunes.shuffle", shuffle ? "1" : "0");
        updateToggles();
    });
    getElement("repeat").addEventListener("click", () => {
        repeat = repeat === "off" ? "all" : repeat === "all" ? "one" : "off";
        savePref("tunes.repeat", repeat);
        updateToggles();
    });

    const vol = getElement<HTMLInputElement>("vol");
    audio.volume = Number(vol.value) / 100;
    vol.addEventListener("input", () => {
        audio.volume = Number(vol.value) / 100;
    });

    const seek = getElement("seek");
    seek.addEventListener("click", (e) => {
        const rect = seek.getBoundingClientRect();
        const ratio = (e.clientX - rect.left) / rect.width;
        if (isFinite(audio.duration)) {
            audio.currentTime = ratio * audio.duration;
        }
    });

    audio.addEventListener("timeupdate", () => {
        getElement("elapsed").textContent = fmtTime(audio.currentTime);
        const pct = audio.duration ? (audio.currentTime / audio.duration) * 100 : 0;
        getElement("seekFill").style.width = `${pct}%`;
    });
    audio.addEventListener("loadedmetadata", () => {
        getElement("duration").textContent = fmtTime(audio.duration);
    });
    audio.addEventListener("play", () => {
        getElement("play").textContent = "||";
    });
    audio.addEventListener("pause", () => {
        getElement("play").textContent = "▶";
    });
    audio.addEventListener("ended", onEnded);
};
