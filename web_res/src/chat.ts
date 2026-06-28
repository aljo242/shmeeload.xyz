import { getElement } from "./dom.js";

const DEFAULT_NAME = "anon";
const DEFAULT_DECODING = "utf-8";

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

let userName = DEFAULT_NAME;
let conn: WebSocket | undefined;

function appendLog(text: string) {
    const log = getElement("log");
    const doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
    const item = document.createElement("div");
    item.className = "logLine";
    // Always render as text, never HTML: messages are broadcast verbatim from
    // other clients (and replayed from history), so innerHTML would be XSS.
    item.textContent = text;
    log.appendChild(item);
    if (doScroll) {
        log.scrollTop = log.scrollHeight - log.clientHeight;
    }
}

function clearLog() {
    getElement("log").textContent = "";
}

// connectRoom switches to a room: it closes any current connection, clears the
// log, opens a new socket scoped to the room, and announces the join.
function connectRoom(room: string) {
    if (conn) {
        conn.onclose = null; // a deliberate switch should not log "disconnected"
        conn.close();
    }
    clearLog();

    document.querySelectorAll<HTMLButtonElement>("#rooms button").forEach((b) => {
        b.classList.toggle("active", b.dataset.room === room);
    });

    const c = new WebSocket(
        `${websocketPrefix}${document.location.host}/chat/ws?room=${encodeURIComponent(room)}`,
    );
    c.binaryType = "arraybuffer";
    c.onmessage = (evt) => {
        appendLog(typeof evt.data === "string" ? evt.data : decode(evt.data as ArrayBuffer));
    };
    c.onopen = () => c.send(encode(`${userName} joined.`));
    c.onclose = () => appendLog("— disconnected —");
    c.onerror = () => appendLog("— connection error —");
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
        if (conn === undefined || conn.readyState !== WebSocket.OPEN || msg.value === "") {
            return;
        }
        conn.send(encode(`${userName}: ${msg.value}`));
        msg.value = "";
    });
};
