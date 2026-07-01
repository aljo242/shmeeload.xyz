import { getElement } from "./dom.js";
import { initMiniPlayer } from "./miniplayer.js";
import { initSparkles } from "./sparkles.js";
import { initTheme } from "./theme.js";

const DEFAULT_NAME = "anon";
const DEFAULT_DECODING = "utf-8";

// A frame starting with this NUL byte is a presence (roster) update, not a chat
// line; the server never puts a NUL in a chat line.
const ROSTER_PREFIX = 0;

// Use the page's own scheme to pick the matching websocket scheme.
const websocketPrefix = window.location.protocol === "https:" ? "wss://" : "ws://";

const encoder = new TextEncoder();
const decoder = new TextDecoder(DEFAULT_DECODING);

function encode(msg: string): ArrayBuffer {
    return encoder.encode(msg).buffer as ArrayBuffer;
}

function decode(buf: ArrayBuffer): string {
    return decoder.decode(buf);
}

const urlPattern = /https?:\/\/[^\s]+/g;

let userName = DEFAULT_NAME;
let currentRoom = "";
let conn: WebSocket | undefined;

// The message log is capped so a long-lived tab can't grow the DOM without
// bound, and appends are coalesced into one reflow per animation frame so a
// burst (history replay on join, or a busy room) renders in a single batch.
const MAX_LOG_LINES = 500;
let pendingLines: HTMLDivElement[] = [];
let flushScheduled = false;

// buildLine renders one line, turning URLs into safe anchor elements. It never
// uses innerHTML: messages are untrusted (from other clients and history).
function buildLine(text: string, cls: string): HTMLDivElement {
    const item = document.createElement("div");
    item.className = cls;

    let last = 0;
    for (const match of text.matchAll(urlPattern)) {
        const start = match.index ?? 0;
        if (start > last) {
            item.appendChild(document.createTextNode(text.slice(last, start)));
        }
        const a = document.createElement("a");
        a.href = match[0]; // only http(s) URLs match, so this is safe
        a.textContent = match[0];
        a.target = "_blank";
        a.rel = "noopener noreferrer";
        item.appendChild(a);
        last = start + match[0].length;
    }
    if (last < text.length) {
        item.appendChild(document.createTextNode(text.slice(last)));
    }
    return item;
}

// flushLog appends every queued line in one batch, trims the log to the cap,
// and keeps the view pinned to the bottom if it already was.
function flushLog() {
    flushScheduled = false;
    if (pendingLines.length === 0) {
        return;
    }
    const log = getElement("log");
    // Checked before the append, which changes scrollHeight.
    const doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;

    const frag = document.createDocumentFragment();
    for (const item of pendingLines) {
        frag.appendChild(item);
    }
    pendingLines = [];
    log.appendChild(frag);

    while (log.childElementCount > MAX_LOG_LINES && log.firstChild) {
        log.removeChild(log.firstChild);
    }
    if (doScroll) {
        log.scrollTop = log.scrollHeight - log.clientHeight;
    }
}

function appendLog(text: string, cls = "logLine") {
    pendingLines.push(buildLine(text, cls));
    // Bound the queue too: rAF does not fire while the tab is backgrounded, so a
    // flood there must not accumulate past what the log would keep anyway.
    if (pendingLines.length > MAX_LOG_LINES) {
        pendingLines.splice(0, pendingLines.length - MAX_LOG_LINES);
    }
    if (!flushScheduled) {
        flushScheduled = true;
        requestAnimationFrame(flushLog);
    }
}

function renderRoster(json: string) {
    let names: string[] = [];
    try {
        names = JSON.parse(json) as string[];
    } catch {
        return;
    }
    const el = getElement("roster");
    el.textContent = names.length > 0 ? `online (${names.length}): ${names.join(", ")}` : "online (0)";
}

function clearLog() {
    pendingLines = []; // drop lines queued from the previous room
    getElement("log").textContent = "";
}

function connectRoom(room: string) {
    if (conn) {
        conn.onclose = null; // a deliberate switch should not log "disconnected"
        conn.close();
    }
    currentRoom = room;
    clearLog();
    getElement("roster").textContent = "";

    document.querySelectorAll<HTMLButtonElement>("#rooms button").forEach((b) => {
        b.classList.toggle("active", b.dataset.room === room);
    });

    const url =
        `${websocketPrefix}${document.location.host}/chat/ws` +
        `?room=${encodeURIComponent(room)}&name=${encodeURIComponent(userName)}`;
    const c = new WebSocket(url);
    c.binaryType = "arraybuffer";
    c.onmessage = (evt) => {
        const text = typeof evt.data === "string" ? evt.data : decode(evt.data as ArrayBuffer);
        if (text.charCodeAt(0) === ROSTER_PREFIX) {
            renderRoster(text.slice(1));
            return;
        }
        appendLog(text);
    };
    c.onclose = () => appendLog("— disconnected —", "statusLine");
    c.onerror = () => appendLog("— connection error —", "statusLine");
    conn = c;
}

async function loadRooms(): Promise<string[]> {
    try {
        const resp = await fetch("/chat/rooms");
        const rooms = (await resp.json()) as string[];
        return Array.isArray(rooms) ? rooms : [];
    } catch {
        return [];
    }
}

function openPopUpForm() {
    getElement("popUpForm").style.display = "block";
    getElement<HTMLInputElement>("chatname").focus();
}

function closePopUpForm() {
    getElement("popUpForm").style.display = "none";
    getElement<HTMLInputElement>("msg").focus();
}

window.onload = async () => {
    initTheme(); // shmee-man button cycles themes, same as the other pages
    void initMiniPlayer(); // "now playing" bar if a track is in progress
    initSparkles();

    if (!("WebSocket" in window)) {
        appendLog("Your browser does not support WebSockets.");
        return;
    }

    const rooms = await loadRooms();
    const roomsEl = getElement("rooms");
    for (const room of rooms) {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "roomBtn";
        btn.textContent = room;
        btn.dataset.room = room;
        btn.addEventListener("click", () => connectRoom(room));
        roomsEl.appendChild(btn);
    }

    openPopUpForm();

    // Handle submit (not the button's click) so pressing Enter in the name field
    // runs this instead of the form's native submission, which reloads the page.
    getElement("signInForm").addEventListener("submit", (evt) => {
        evt.preventDefault();
        const userNameInput = getElement<HTMLInputElement>("chatname");
        userName = userNameInput.value === "" ? DEFAULT_NAME : userNameInput.value;
        closePopUpForm();
        if (rooms.length > 0) {
            connectRoom(rooms[0]); // auto-join the first room
        }
    });

    getElement("send_msg_form").addEventListener("submit", (evt) => {
        evt.preventDefault();
        const msg = getElement<HTMLInputElement>("msg");
        const text = msg.value;
        if (text === "") {
            return;
        }
        // "/nick <name>" changes the display name by reconnecting to the room.
        if (text.startsWith("/nick ")) {
            const newName = text.slice("/nick ".length).trim();
            if (newName !== "" && currentRoom !== "") {
                userName = newName;
                connectRoom(currentRoom);
            }
            msg.value = "";
            return;
        }
        if (conn === undefined || conn.readyState !== WebSocket.OPEN) {
            return;
        }
        conn.send(encode(text)); // raw text; the server stamps time + name
        msg.value = "";
    });
};
