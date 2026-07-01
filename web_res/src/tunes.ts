import { getElement } from "./dom.js";
import { loadState, saveState } from "./playerstate.js";
import { initSparkles } from "./sparkles.js";
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
let vizMode: "scope" | "bars" = "bars";
let restoreTime = 0; // position to seek to once metadata loads (resume handoff)
let lastSaved = -2; // throttle for persisting the play position
const audio = new Audio();
audio.preload = "none";

// Web Audio graph for the waveform visualizer, built lazily on the first user
// gesture (so the AudioContext can start) and shared for the page's lifetime.
let audioCtx: AudioContext | undefined;
let analyser: AnalyserNode | undefined;
let vizData: Uint8Array<ArrayBuffer> | undefined;
const BARS = 56;
const barHeights = new Array<number>(BARS).fill(0); // for smooth bar decay
const DOWNSAMPLE = 1.5; // render small and upscale (pixelated) for a low-res look
let idleT = 0; // animation phase for the idle (pre-play) waveform
let signalLevel = 0; // smoothed overlay gate (0..1) from the frequency buckets
let freqData: Uint8Array<ArrayBuffer> | undefined; // spectrum for the gate
// Per-bucket thresholds for the scope overlay gate: each band triggers at its
// own sensitivity (bass is loud, so it needs a higher bar to count).
const FREQ_BUCKETS = [
    { lo: 0, hi: 24, threshold: 0.18 },   // bass: most sensitive
    { lo: 24, hi: 80, threshold: 0.3 },
    { lo: 80, hi: 180, threshold: 0.42 },
    { lo: 180, hi: 360, threshold: 0.5 }, // treble: must be loud to count
];
let noiseCanvas: HTMLCanvasElement | undefined;
let noiseImg: ImageData | undefined;
let signalCanvas: HTMLCanvasElement | undefined; // trace/bars only (no static)

// drawSnow paints a frame of CRT static at the (downsampled) buffer resolution,
// so the screen grain degrades at the same low res as the trace.
function drawSnow(ctx: CanvasRenderingContext2D, w: number, h: number, alpha: number) {
    if (noiseCanvas === undefined) {
        noiseCanvas = document.createElement("canvas");
    }
    const nctx = noiseCanvas.getContext("2d");
    if (nctx === null) {
        return;
    }
    if (noiseCanvas.width !== w || noiseCanvas.height !== h || noiseImg === undefined) {
        noiseCanvas.width = w;
        noiseCanvas.height = h;
        noiseImg = nctx.createImageData(w, h);
    }
    const d = noiseImg.data;
    for (let i = 0; i < d.length; i += 4) {
        const v = Math.floor(Math.random() * 255);
        d[i] = v;
        d[i + 1] = v;
        d[i + 2] = Math.min(255, v + 14); // faintly blue, like a dead channel
        d[i + 3] = 255;
    }
    nctx.putImageData(noiseImg, 0, 0);
    ctx.globalAlpha = alpha;
    ctx.drawImage(noiseCanvas, 0, 0);
    ctx.globalAlpha = 1;
}

// drawViz renders the visualizer each frame. The signal (trace/bars + ghost +
// glitch) is drawn on its own canvas with no static, so it can be projected
// cleanly to the full-page overlay; the screen is then static + signal.
function drawViz() {
    requestAnimationFrame(drawViz);
    if (vizData === undefined) {
        return;
    }
    idleT += 0.05;
    const viz = getElement<HTMLCanvasElement>("viz");
    const vizCtx = viz.getContext("2d");
    if (vizCtx === null) {
        return;
    }
    // Downsample: draw into a small buffer and let CSS upscale it (pixelated)
    // for a chunky low-res look.
    const w = Math.max(1, Math.floor(viz.clientWidth / DOWNSAMPLE));
    const h = Math.max(1, Math.floor(viz.clientHeight / DOWNSAMPLE));
    if (viz.width !== w) {
        viz.width = w;
    }
    if (viz.height !== h) {
        viz.height = h;
    }

    if (signalCanvas === undefined) {
        signalCanvas = document.createElement("canvas");
    }
    const sig = signalCanvas;
    if (sig.width !== w || sig.height !== h) {
        sig.width = w;
        sig.height = h;
    }
    const ctx = sig.getContext("2d");
    if (ctx === null) {
        return;
    }
    ctx.imageSmoothingEnabled = false;
    const color = getComputedStyle(document.documentElement).getPropertyValue("--scheme-link").trim() || "#14ff14";
    ctx.globalAlpha = 1;

    // tear copies a random horizontal band and re-pastes it shifted: the
    // occasional CRT signal tear.
    const tear = (prob: number, maxShift: number) => {
        if (Math.random() < prob) {
            const sy = Math.random() * h * 0.8;
            const sh = 1 + Math.random() * h * 0.3;
            ctx.drawImage(sig, 0, sy, w, sh, (Math.random() - 0.5) * maxShift, sy, w, sh);
        }
    };

    if (vizMode === "bars") {
        ctx.clearRect(0, 0, w, h);
        if (analyser !== undefined) {
            analyser.getByteFrequencyData(vizData);
        } else {
            for (let i = 0; i < vizData.length; i++) {
                vizData[i] = Math.max(0, Math.round((Math.sin(i * 0.18 + idleT) * 0.5 + 0.5) * 22 - i * 0.04));
            }
        }
        ctx.shadowBlur = 5;
        ctx.shadowColor = color;
        ctx.fillStyle = color;
        const bw = w / BARS;
        for (let i = 0; i < BARS; i++) {
            const target = vizData[i * 4] / 255; // spread across the low-mid spectrum
            // rise immediately, fall back slowly for a smooth decay
            barHeights[i] = target > barHeights[i] ? target : barHeights[i] * 0.965;
            const bh = Math.max(1, barHeights[i] * h);
            ctx.fillRect(i * bw, h - bh, Math.max(1, bw - 1), bh);
        }
        tear(0.04, 12);
    } else {
        // Oscilloscope with phosphor ghosting: fade the previous trace toward
        // transparent so it leaves a glowing trail.
        if (analyser !== undefined) {
            analyser.getByteTimeDomainData(vizData);
            // Per-bucket frequency gate: the overlay reacts to whichever band is
            // most over its own threshold.
            if (freqData !== undefined) {
                analyser.getByteFrequencyData(freqData);
                let gate = 0;
                for (const b of FREQ_BUCKETS) {
                    let sum = 0;
                    for (let i = b.lo; i < b.hi; i++) {
                        sum += freqData[i];
                    }
                    const level = sum / ((b.hi - b.lo) * 255);
                    gate = Math.max(gate, (level - b.threshold) / 0.2);
                }
                // Fast attack, slow release: a bass hit pops the overlay up and
                // it trails off afterward (a bigger hit = a longer tail).
                const target = Math.min(1, Math.max(0, gate));
                signalLevel = target > signalLevel ? signalLevel * 0.35 + target * 0.65 : signalLevel * 0.95;
            }
        } else {
            for (let i = 0; i < vizData.length; i++) {
                vizData[i] = 128 + Math.round(Math.sin(i * 0.05 + idleT) * 7);
            }
            signalLevel *= 0.9;
        }
        ctx.globalCompositeOperation = "destination-out";
        ctx.fillStyle = "rgba(0, 0, 0, 0.045)"; // lower alpha = longer trail
        ctx.fillRect(0, 0, w, h);
        ctx.globalCompositeOperation = "source-over";

        ctx.lineWidth = 2.2;
        ctx.shadowBlur = 7;
        ctx.shadowColor = color;
        ctx.strokeStyle = color;
        ctx.beginPath();
        const slice = w / vizData.length;
        for (let i = 0; i < vizData.length; i++) {
            const x = i * slice;
            const y = (vizData[i] / 128) * (h / 2);
            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        }
        ctx.stroke();
        tear(0.05, 14);
    }

    // Compose the screen: static underneath, the signal on top.
    vizCtx.imageSmoothingEnabled = false;
    vizCtx.clearRect(0, 0, w, h);
    drawSnow(vizCtx, w, h, vizMode === "bars" ? 0.07 : 0.06);
    vizCtx.drawImage(sig, 0, 0);
}

// setupViz wires the audio element through an analyser. Must be called from a
// user gesture so the context starts (and the audio keeps making sound). Routing
// the element through the context is one-shot, so it is guarded.
function setupViz() {
    if (audioCtx !== undefined) {
        void audioCtx.resume();
        return;
    }
    audioCtx = new AudioContext();
    const source = audioCtx.createMediaElementSource(audio);
    analyser = audioCtx.createAnalyser();
    analyser.fftSize = 1024;
    analyser.smoothingTimeConstant = 0.85; // smooth the frequency data for gentler bars
    source.connect(analyser);
    analyser.connect(audioCtx.destination);
    vizData = new Uint8Array(analyser.fftSize);
    freqData = new Uint8Array(analyser.frequencyBinCount);
    void audioCtx.resume();
}

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
    saveState({ file: tunes[i].file });
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
        const v = localStorage.getItem("tunes.viz");
        if (v === "scope" || v === "bars") {
            vizMode = v;
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
        li.addEventListener("click", () => {
            setupViz();
            load(i);
        });
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

    player.addEventListener("pointerdown", (e) => {
        // Grab the metal/LCD chrome to move it; clicking a control does not drag.
        const target = e.target as Element | null;
        if (target && target.closest("button, input, #seek, #playlist")) {
            return;
        }
        dragging = true;
        const rect = player.getBoundingClientRect();
        offX = e.clientX - rect.left;
        offY = e.clientY - rect.top;
        player.setPointerCapture(e.pointerId);
        e.preventDefault();
    });
    player.addEventListener("pointermove", (e) => {
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
    player.addEventListener("pointerup", end);
    player.addEventListener("pointercancel", end);
}

// projectLoop mirrors the scope onto a faint full-page overlay, so the whole
// tunes page gets a subtle CRT wash (opacity set in CSS).
function projectLoop() {
    requestAnimationFrame(projectLoop);
    if (signalCanvas === undefined) {
        return; // nothing playing yet
    }
    const overlay = getElement<HTMLCanvasElement>("vizOverlay");
    const octx = overlay.getContext("2d");
    if (octx === null) {
        return;
    }
    const w = overlay.clientWidth;
    const h = overlay.clientHeight;
    if (w === 0 || h === 0) {
        return;
    }
    if (overlay.width !== w) {
        overlay.width = w;
    }
    if (overlay.height !== h) {
        overlay.height = h;
    }
    octx.clearRect(0, 0, w, h);
    octx.imageSmoothingEnabled = false;
    if (vizMode === "bars") {
        octx.drawImage(signalCanvas, 0, 0, w, h); // bars fill the page, not zoomed
    } else {
        // Volume-gated per frequency bucket: signalLevel is already the gated
        // ramp (0 below all thresholds). Zoomed in + centered.
        const a = Math.min(1, Math.max(0, signalLevel));
        if (a > 0.01) {
            octx.globalAlpha = a;
            const zoom = 2;
            const dw = w * zoom;
            const dh = h * zoom;
            octx.drawImage(signalCanvas, (w - dw) / 2, (h - dh) / 2, dw, dh);
            octx.globalAlpha = 1;
        }
    }
}

window.onload = async () => {
    initTheme();
    initSparkles(10); // fewer here so the player stays the focus
    loadPrefs();
    updateToggles();
    makeDraggable();
    vizData = new Uint8Array(1024); // run the visualizer (idle) before play
    freqData = new Uint8Array(1024);
    requestAnimationFrame(drawViz);
    requestAnimationFrame(projectLoop);
    // Connect the analyser on the first interaction anywhere (the AudioContext
    // can't start without a gesture), so the visualizer comes alive even when you
    // arrive with a track already playing from the mini-player.
    const kick = () => setupViz();
    window.addEventListener("pointerdown", kick, { once: true });
    window.addEventListener("keydown", kick, { once: true });
    await loadList();

    getElement("play").addEventListener("click", () => {
        setupViz();
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
    getElement("prev").addEventListener("click", () => {
        setupViz();
        prev();
    });
    getElement("next").addEventListener("click", () => {
        setupViz();
        next();
    });
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
    getElement("copyLink").addEventListener("click", () => {
        const file = current >= 0 ? tunes[current].file : tunes.length > 0 ? tunes[0].file : "";
        if (file === "") {
            return;
        }
        const url = `${window.location.origin}/tunes/home?track=${encodeURIComponent(file)}`;
        const btn = getElement("copyLink");
        void navigator.clipboard.writeText(url).then(() => {
            btn.textContent = "copied";
            window.setTimeout(() => {
                btn.textContent = "link";
            }, 1200);
        }).catch(() => undefined);
    });
    const vizModeBtn = getElement("vizMode");
    vizModeBtn.textContent = vizMode;
    vizModeBtn.addEventListener("click", () => {
        setupViz();
        vizMode = vizMode === "scope" ? "bars" : "scope";
        vizModeBtn.textContent = vizMode;
        savePref("tunes.viz", vizMode);
    });

    const vol = getElement<HTMLInputElement>("vol");
    audio.volume = Number(vol.value) / 100;
    vol.addEventListener("input", () => {
        audio.volume = Number(vol.value) / 100;
        saveState({ vol: Number(vol.value) });
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
        if (Math.abs(audio.currentTime - lastSaved) >= 1) {
            lastSaved = audio.currentTime;
            saveState({ time: audio.currentTime });
        }
    });
    audio.addEventListener("loadedmetadata", () => {
        getElement("duration").textContent = fmtTime(audio.duration);
        if (restoreTime > 0 && isFinite(audio.duration)) {
            audio.currentTime = Math.min(restoreTime, audio.duration - 0.1);
            restoreTime = 0;
        }
    });
    audio.addEventListener("play", () => {
        getElement("play").textContent = "||";
        saveState({ playing: true });
    });
    audio.addEventListener("pause", () => {
        getElement("play").textContent = "▶";
        saveState({ playing: false });
    });
    audio.addEventListener("ended", onEnded);
    window.addEventListener("beforeunload", () => saveState({ time: audio.currentTime }));

    // A shared ?track= link preselects and starts that song; otherwise resume
    // where a previous visit (or the mini-player) left off.
    const sharedTrack = new URLSearchParams(window.location.search).get("track");
    const sharedIdx = sharedTrack === null ? -1 : tunes.findIndex((t) => t.file === sharedTrack);
    if (sharedIdx >= 0) {
        load(sharedIdx, true); // a browser may still block autoplay until you click
    } else {
        const saved = loadState();
        if (saved !== null) {
            const i = tunes.findIndex((t) => t.file === saved.file);
            if (i >= 0) {
                vol.value = String(saved.vol);
                audio.volume = saved.vol / 100;
                restoreTime = saved.time;
                load(i, saved.playing);
            }
        }
    }
};
