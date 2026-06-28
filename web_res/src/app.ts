import { initTheme } from "./theme.js";

// No service worker is registered: the previous one was empty, so it added
// install/update churn and controlled scope "/" with no offline benefit.
// Reintroduce a registration here only alongside a real caching worker.

initTheme();
