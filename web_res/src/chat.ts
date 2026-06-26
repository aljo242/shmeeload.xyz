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

class User {
    userName: string;
    conn: WebSocket;

    constructor(name: string, conn: WebSocket) {
        this.userName = name;
        this.conn = conn;
        this.signIn();
    }

    signIn() {
        this.broadcast(encode(`${this.userName} signed in.`));
    }

    broadcast(buf: ArrayBuffer) {
        // Sending on a closing/closed socket throws; skip if it isn't open.
        if (this.conn.readyState === WebSocket.OPEN) {
            this.conn.send(buf);
        }
    }
}

function openPopUpForm() {
    getElement("popUpForm").style.display = "block";
    // Move focus into the dialog so keyboard/AT users start inside it.
    getElement<HTMLInputElement>("chatname").focus();
}

function closePopUpForm() {
    getElement("popUpForm").style.display = "none";
    // Return focus to the message input now that the dialog is dismissed.
    getElement<HTMLInputElement>("msg").focus();
}

function appendLog(item: HTMLDivElement) {
    const log = getElement("log");
    const doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
    log.appendChild(item);
    if (doScroll) {
        log.scrollTop = log.scrollHeight - log.clientHeight;
    }
}

window.onload = () => {
    openPopUpForm();
    const msg = getElement<HTMLInputElement>("msg");

    if (!("WebSocket" in window)) {
        const item = document.createElement("div");
        item.textContent = "Your browser does not support WebSockets.";
        appendLog(item);
        return;
    }

    const conn = new WebSocket(websocketPrefix + document.location.host + "/chat/ws");
    conn.binaryType = "arraybuffer";

    conn.onmessage = (evt) => {
        const item = document.createElement("div");
        // The hub relays messages as text frames, so evt.data is a string; decode
        // only if a binary frame ever arrives. Always render as text, never HTML:
        // messages are broadcast verbatim from other clients, so innerHTML would
        // be a stored XSS.
        item.textContent =
            typeof evt.data === "string" ? evt.data : decode(evt.data as ArrayBuffer);
        appendLog(item);
    };

    conn.onclose = () => {
        const item = document.createElement("div");
        item.textContent = "Connection to server closed.";
        appendLog(item);
    };

    conn.onerror = () => {
        const item = document.createElement("div");
        item.textContent = "Could not connect to the chat server.";
        appendLog(item);
    };

    let user: User | undefined;

    getElement("signInButton").addEventListener("click", () => {
        const userNameInput = getElement<HTMLInputElement>("chatname");
        if (userNameInput.value === "") {
            userNameInput.value = DEFAULT_NAME;
        }
        user = new User(userNameInput.value, conn);
        closePopUpForm();
    });

    getElement("send_msg_form").addEventListener("submit", (evt) => {
        evt.preventDefault();
        if (user === undefined || msg.value === "") {
            return;
        }
        user.broadcast(encode(`${user.userName}: ${msg.value}`));
        msg.value = "";
    });
};
