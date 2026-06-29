import { getElement } from "./dom.js";
import { initTheme } from "./theme.js";

interface Tune {
    file: string;
    name: string;
    size: number;
}

let tunes: Tune[] = [];
let current = -1;
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

function next() {
    load(current >= tunes.length - 1 ? 0 : current + 1);
}

function prev() {
    load(current <= 0 ? tunes.length - 1 : current - 1);
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

window.onload = async () => {
    initTheme();
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
    audio.addEventListener("ended", next);
};
