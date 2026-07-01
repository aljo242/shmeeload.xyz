// Shared playback state, persisted in localStorage so the full player (tunes
// page) and the mini-player (every other page) hand off across navigations.

export type RepeatMode = "off" | "all" | "one";

export interface PlayerState {
    file: string; // current track filename ("" if none)
    time: number; // position in seconds
    playing: boolean;
    vol: number; // 0-100
    shuffle: boolean;
    repeat: RepeatMode;
}

const KEY = {
    file: "tunes.curfile",
    time: "tunes.time",
    playing: "tunes.playing",
    vol: "tunes.vol",
    shuffle: "tunes.shuffle",
    repeat: "tunes.repeat",
};

export function loadState(): PlayerState | null {
    try {
        const file = localStorage.getItem(KEY.file);
        if (file === null || file === "") {
            return null;
        }
        const repeat = localStorage.getItem(KEY.repeat);
        return {
            file,
            time: Number(localStorage.getItem(KEY.time) ?? "0") || 0,
            playing: localStorage.getItem(KEY.playing) === "1",
            vol: Number(localStorage.getItem(KEY.vol) ?? "80") || 80,
            shuffle: localStorage.getItem(KEY.shuffle) === "1",
            repeat: repeat === "all" || repeat === "one" ? repeat : "off",
        };
    } catch {
        return null;
    }
}

export function saveState(partial: Partial<PlayerState>) {
    try {
        if (partial.file !== undefined) {
            localStorage.setItem(KEY.file, partial.file);
        }
        if (partial.time !== undefined) {
            localStorage.setItem(KEY.time, String(Math.floor(partial.time)));
        }
        if (partial.playing !== undefined) {
            localStorage.setItem(KEY.playing, partial.playing ? "1" : "0");
        }
        if (partial.vol !== undefined) {
            localStorage.setItem(KEY.vol, String(Math.round(partial.vol)));
        }
        if (partial.shuffle !== undefined) {
            localStorage.setItem(KEY.shuffle, partial.shuffle ? "1" : "0");
        }
        if (partial.repeat !== undefined) {
            localStorage.setItem(KEY.repeat, partial.repeat);
        }
    } catch {
        // localStorage may be unavailable (private mode); state just won't persist.
    }
}
