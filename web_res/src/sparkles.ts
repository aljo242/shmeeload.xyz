// Scatters twinkling gold sparkles across the page at random spots. Each one
// fades out at the end of its twinkle cycle and relocates, so the layout is
// random on every load and keeps drifting the whole time.

const VARIANTS = ["sparkle.png", "sparkle2.png", "sparkle3.png", "sparkle4.png"];

function place(img: HTMLImageElement) {
    img.src = "/static/img/" + VARIANTS[Math.floor(Math.random() * VARIANTS.length)];
    img.style.top = (4 + Math.random() * 86).toFixed(1) + "%";
    img.style.left = (3 + Math.random() * 90).toFixed(1) + "%";
    img.style.width = (36 + Math.random() * 54).toFixed(0) + "px";
    img.style.setProperty("--op", (0.5 + Math.random() * 0.4).toFixed(2));
    const dur = 5 + Math.random() * 15; // 5s (fastest) .. 20s (slowest)
    img.style.animationDuration = dur.toFixed(1) + "s";
    img.style.animationDelay = "-" + (Math.random() * dur).toFixed(1) + "s"; // random phase
    // Random fade profile: gentle in, but the fade-out length varies per sparkle.
    img.style.animationName = ["twinkle", "twinkleLinger", "twinkleQuick"][Math.floor(Math.random() * 3)];
}

export function initSparkles(count = 14) {
    for (let i = 0; i < count; i++) {
        const img = document.createElement("img");
        img.className = "flare";
        img.alt = "";
        place(img);
        // At the end of each twinkle cycle it's fully faded, so relocating then
        // is invisible: it just reappears somewhere new.
        img.addEventListener("animationiteration", () => place(img));
        document.body.appendChild(img);
    }
}
