import { initMiniPlayer } from "./miniplayer.js";
import { initSparkles } from "./sparkles.js";
import { initTheme } from "./theme.js";

// No service worker is registered: the previous one was empty, so it added
// install/update churn and controlled scope "/" with no offline benefit.
// Reintroduce a registration here only alongside a real caching worker.

initTheme();
void initMiniPlayer(); // shows the "now playing" bar if a track is in progress
initSparkles();
