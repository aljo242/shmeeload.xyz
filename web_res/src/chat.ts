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

// appendLog renders one line, turning URLs into safe anchor elements. It never
// uses innerHTML: messages are untrusted (from other clients and history).
function appendLog(text: string, cls = "logLine") {
    const log = getElement("log");
    const doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
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

    log.appendChild(item);
    if (doScroll) {
        log.scrollTop = log.scrollHeight - log.clientHeight;
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

    getElement("signInButton").addEventListener("click", () => {
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
