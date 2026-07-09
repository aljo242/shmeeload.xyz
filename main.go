package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aljo242/shmeeload.xyz/handlers"
	"github.com/aljo242/shmeeload.xyz/internal/log"
	"github.com/quic-go/quic-go/http3"

	_ "time/tzdata" // embed the zoneinfo DB so the display timezone resolves in the minimal container
)

const (
	// DefaultConfigFile is the default path to the JSON configuration file.
	DefaultConfigFile string = "sample/sample_config.json"

	// shutdownTimeout bounds how long graceful shutdown waits for in-flight requests.
	shutdownTimeout = 10 * time.Second
)

// securityTxt is the vulnerability-disclosure contact served at
// /.well-known/security.txt (RFC 9116). Update Expires before it lapses.
const securityTxt = `Contact: mailto:info@djinntek.space
Expires: 2028-01-01T00:00:00Z
Preferred-Languages: en
Canonical: https://djinntek.space/.well-known/security.txt
`

var (
	configFile string
	devMode    bool
)

func init() {
	flag.StringVar(&configFile, "c", DefaultConfigFile, "Full path to JSON configuration file")
	flag.BoolVar(&devMode, "dev", false, "dev mode: serve site/ from disk (no embed/minify) for local preview")
}

// fatal logs an error and exits. Kept out of functions with deferred cleanup so
// it never skips a defer.
func fatal(msg string, err error) {
	log.Error(msg, "err", err)
	os.Exit(1)
}

// buildRouter wires the embedded static site, the chat websocket, and the donate
// endpoint behind shared middleware. Static assets (/static/*, /files/*,
// /manifest.json, …) are served straight from the embedded FS; the few pretty
// page URLs map to their HTML file, and legacy placeholders redirect.
//
// Routing uses the stdlib ServeMux: GET patterns match GET and HEAD (other
// methods get a 405), "/{$}" matches the root exactly, and "/" is the
// least-specific catch-all for everything else.
func buildRouter(cfg Config, hub *Hub, site *staticSite) http.Handler {
	mux := http.NewServeMux()

	// In dev mode, site/ is served straight from disk (fresh each request, no
	// embed/minify/compress) so HTML/CSS/JS edits show on a browser refresh
	// without rebuilding. no-cache forces revalidation so the browser never holds
	// a stale asset mid-iteration. Production serves the optimized embedded site.
	devStatic := http.FileServer(http.Dir("site"))
	devServe := func(w http.ResponseWriter, rq *http.Request, h http.Handler) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, rq)
	}

	// serveAsset serves a named asset, 404ing if absent.
	serveAsset := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, rq *http.Request) {
			if cfg.Dev {
				devServe(w, rq, http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
					http.ServeFile(w, rq, filepath.Join("site", name))
				}))
				return
			}
			if !site.serve(w, rq, name) {
				http.NotFound(w, rq)
			}
		}
	}
	page := serveAsset
	visits := newVisitCounter(filepath.Join(filepath.Dir(chatDBPathOf(cfg)), "gamers-visits"))
	mcStat := newStatusCache(cfg.MCServerAddr)
	heads := newMCHeadProxy()
	live := newLiveStore()
	pihole := newPiholeClient(cfg.PiholeURL, cfg.PiholePassword)
	hosts := newHostStore()
	metrics, err := newMetricsStore(metricsDBPathOf(cfg))
	if err != nil {
		log.Error("metrics history disabled", "err", err)
	}
	go metrics.run(context.Background(), pihole, hosts) // nil-safe; unmanaged like the other pollers
	cert := newCertChecker(apexDomain(cfg.Domains))
	underConstruction := func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, "/under-construction", http.StatusTemporaryRedirect)
	}

	// dynamic endpoints
	conns := newConnLimiter(wsMaxPerIP, wsMaxTotal)
	rooms := chatRoomsOf(cfg)
	mux.HandleFunc("GET /donate/{cryptoname}", handlers.DonateHandler(cfg.CacheMaxAge))
	mux.HandleFunc("GET /chat/ws", serveWs(hub, conns, roomSet(rooms)))
	mux.HandleFunc("GET /chat/rooms", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(rooms)
	})
	tunesDir := tunesDirOf(cfg)
	mux.HandleFunc("GET /tunes/list", tunesListHandler(tunesDir))
	mux.HandleFunc("GET /tunes/file/{name}", tunesFileHandler(tunesDir))

	// pages (pretty URL -> embedded HTML)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, "/home", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("GET /home", page("home.html"))
	mux.HandleFunc("GET /resume/home", page("resume.html"))
	mux.HandleFunc("GET /hall-of-art/home", page("shadow.html"))
	mux.HandleFunc("GET /chat/home", page("chat.html"))
	mux.HandleFunc("GET /tunes/home", page("tunes.html"))
	mux.HandleFunc("GET /shop/home", page("shop.html"))
	mux.HandleFunc("GET /gamers", func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, "/gamers/home", http.StatusPermanentRedirect)
	})
	gamersPage := page("gamers.html")
	mux.HandleFunc("GET /gamers/home", func(w http.ResponseWriter, rq *http.Request) {
		visits.Inc()
		gamersPage(w, rq)
	})
	mux.HandleFunc("GET /gamers/count", func(w http.ResponseWriter, rq *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(map[string]int64{"visits": visits.Get()})
	})
	mux.HandleFunc("GET /gamers/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(mcStat.get())
	})
	// Merged live status: the fresh SLP fields plus the last pushed telemetry
	// (TPS, in-game day, uptime, whitelist). Drives the /gamers dashboard and the
	// /status page.
	mux.HandleFunc("GET /gamers/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(buildLive(mcStat.get(), live, time.Now()))
	})
	// Ingest for the game host's push. Registered only when a token is set; the
	// bearer token is compared in constant time and the body is size-capped.
	if cfg.MCPushToken != "" {
		mux.HandleFunc("POST /gamers/live", func(w http.ResponseWriter, rq *http.Request) {
			const prefix = "Bearer "
			auth := rq.Header.Get("Authorization")
			token, ok := strings.CutPrefix(auth, prefix)
			if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.MCPushToken)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var p pushedStatus
			dec := json.NewDecoder(io.LimitReader(rq.Body, 8<<10))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&p); err != nil || !p.valid() {
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			live.ingest(p, time.Now())
			w.WriteHeader(http.StatusNoContent)
		})
		// Host stats ingest for the internal dashboard: each box (foundry, the Pi)
		// pushes its own report. Same bearer token; the data is only ever rendered
		// on the Basic-Auth-gated /internal page.
		mux.HandleFunc("POST /internal/host", func(w http.ResponseWriter, rq *http.Request) {
			token, ok := strings.CutPrefix(rq.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.MCPushToken)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var r hostReport
			dec := json.NewDecoder(io.LimitReader(rq.Body, 16<<10))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&r); err != nil || !r.valid() {
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			hosts.ingest(r, time.Now())
			w.WriteHeader(http.StatusNoContent)
		})
	}
	// Player-head avatars, proxied same-origin so the strict img-src CSP holds.
	mux.HandleFunc("GET /gamers/head/{id}", func(w http.ResponseWriter, rq *http.Request) {
		png, ok := heads.get(rq.Context(), rq.PathValue("id"))
		if !ok {
			http.NotFound(w, rq)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(png)
	})
	mux.HandleFunc("GET /under-construction", page("construction.html"))
	mux.HandleFunc("GET /status", page("status.html"))

	// Internal homelab view, gated by Basic Auth. Registered only when
	// credentials are configured; its data (Pi-hole stats) is never on a public
	// endpoint.
	if cfg.InternalUser != "" && cfg.InternalPass != "" {
		// Gated by Basic Auth. The page is server-rendered and refreshed by a
		// <meta refresh>, so page navigations (which do carry Basic Auth, unlike
		// fetch subrequests) are all that is needed. No cookie, token, or script.
		trusted := parseCIDRs(cfg.InternalTrustCIDRs)
		gate := func(h http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, rq *http.Request) {
				// Requests from a trusted internal network skip Basic Auth. The IP
				// is Caddy's X-Real-IP (the true peer), which a client cannot forge
				// since the app is only reachable via Caddy.
				if ipInNets(rq.Header.Get("X-Real-IP"), trusted) {
					h(w, rq)
					return
				}
				u, p, ok := rq.BasicAuth()
				if !ok ||
					subtle.ConstantTimeCompare([]byte(u), []byte(cfg.InternalUser)) != 1 ||
					subtle.ConstantTimeCompare([]byte(p), []byte(cfg.InternalPass)) != 1 {
					w.Header().Set("WWW-Authenticate", `Basic realm="internal"`)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				h(w, rq)
			}
		}
		mux.HandleFunc("GET /internal", gate(func(w http.ResponseWriter, rq *http.Request) {
			now := time.Now()
			certDays, certOK := cert.days()
			// Content negotiation: JSON for API clients, HTML otherwise.
			if strings.Contains(rq.Header.Get("Accept"), "application/json") || rq.URL.Query().Get("format") == "json" {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "no-cache")
				_ = json.NewEncoder(w).Encode(buildInternalAPI(pihole.snapshot(), buildLive(mcStat.get(), live, now), hosts.all(), certDays, certOK, now))
				return
			}
			hist := func(series string) []float64 {
				return metrics.recent(rq.Context(), series, now.Add(-metricsWindow).Unix())
			}
			renderInternal(w, buildInternalView(pihole.snapshot(), buildLive(mcStat.get(), live, now), hosts.all(), hist, certDays, certOK, now))
		}))
	}

	// security.txt for vulnerability-disclosure contact (RFC 9116).
	mux.HandleFunc("GET /.well-known/security.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write([]byte(securityTxt))
	})

	// not-yet-built placeholders
	mux.HandleFunc("GET /chat/signup", underConstruction)
	mux.HandleFunc("GET /chat/signin", underConstruction)

	// everything else is a static asset from the embedded site
	mux.HandleFunc("GET /", func(w http.ResponseWriter, rq *http.Request) {
		if cfg.Dev {
			devServe(w, rq, devStatic)
			return
		}
		if !site.serve(w, rq, rq.URL.Path) {
			http.NotFound(w, rq)
		}
	})

	// Middleware, applied outermost first: security headers, then per-IP rate
	// limiting, then the HTTPS-only Alt-Svc and (opt-in) HSTS headers.
	var h http.Handler = mux
	if cfg.HSTS {
		h = hsts(h)
	}
	if cfg.HTTPS {
		h = altSvc(cfg.Port)(h)
	}
	h = newIPRateLimiter(httpRatePerSec, httpBurst).middleware(h)
	h = redirectToApex(apexDomain(cfg.Domains))(h)
	h = securityHeaders(h)
	return h
}

// parseCIDRs parses a list of CIDR strings, skipping any that are malformed.
func parseCIDRs(cidrs []string) []*net.IPNet {
	var out []*net.IPNet
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// ipInNets reports whether ipStr parses to an IP inside any of nets. An empty
// net list (or unparseable IP) is never trusted.
func ipInNets(ipStr string, nets []*net.IPNet) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// apexDomain returns the first non-www domain (the canonical apex), or "" when
// none is configured.
func apexDomain(domains []string) string {
	for _, d := range domains {
		if !strings.HasPrefix(d, "www.") {
			return d
		}
	}
	return ""
}

// Chat defaults, applied when the config leaves a field empty.
var defaultChatRooms = []string{"general", "music", "art", "tech"}

const (
	defaultChatRetentionDays = 14
	defaultChatDBPath        = "/data/chat.db"
)

func chatRoomsOf(cfg Config) []string {
	if len(cfg.ChatRooms) > 0 {
		return cfg.ChatRooms
	}
	return defaultChatRooms
}

func chatRetentionDaysOf(cfg Config) int {
	if cfg.ChatRetentionDays > 0 {
		return cfg.ChatRetentionDays
	}
	return defaultChatRetentionDays
}

func chatDBPathOf(cfg Config) string {
	if cfg.ChatDBPath != "" {
		return cfg.ChatDBPath
	}
	return defaultChatDBPath
}

// metricsDBPathOf puts the dashboard history DB next to the chat DB.
func metricsDBPathOf(cfg Config) string {
	return filepath.Join(filepath.Dir(chatDBPathOf(cfg)), "metrics.db")
}

func roomSet(rooms []string) map[string]bool {
	m := make(map[string]bool, len(rooms))
	for _, r := range rooms {
		m[r] = true
	}
	return m
}

// runChatCleanup deletes messages past the retention window, once at startup and
// then daily, until ctx is cancelled.
func runChatCleanup(ctx context.Context, store *chatStore, retentionDays int) {
	window := time.Duration(retentionDays) * 24 * time.Hour
	purge := func() {
		if n, err := store.purgeOlderThan(ctx, window); err != nil {
			log.Error("chat cleanup failed", "err", err)
		} else if n > 0 {
			log.Info("chat cleanup", "deleted", n)
		}
	}
	purge()
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			purge()
		}
	}
}

// initServer loads config and returns a configured (but not yet listening)
// http.Server with the chat hub running.
func initServer() (*http.Server, *Hub, Config) {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		fatal("error loading config", err)
	}
	cfg.Dev = cfg.Dev || devMode
	log.Setup(cfg.DebugLog)

	// With ACME on, certmagic manages the certificate; otherwise generate a
	// self-signed one for LAN use.
	if cfg.HTTPS && !cfg.ACME {
		if err := ensureCert(cfg.CertFile, cfg.KeyFile, cfg.TLSHosts); err != nil {
			fatal("error preparing TLS certificate", err)
		}
	}

	// In dev mode the site is served from disk, so skip indexing the embed.
	var site *staticSite
	if !cfg.Dev {
		site, err = newStaticSite(siteFS(), cfg.CacheMaxAge)
		if err != nil {
			fatal("error indexing embedded site", err)
		}
	}

	// Chat persistence is best-effort: if the DB cannot be opened (e.g. /data is
	// missing in a dev run), the chat still works, just without history.
	var store *chatStore
	if s, err := newChatStore(chatDBPathOf(cfg)); err != nil {
		log.Error("chat persistence disabled", "err", err)
	} else {
		store = s
	}

	hub := newHub(store)
	go hub.run()

	srv := &http.Server{
		Addr:              cfg.IP + ":" + cfg.Port,
		Handler:           buildRouter(cfg, hub, site),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		// Generous enough to serve the largest original image to a slow client
		// without truncating; ReadHeaderTimeout is what guards against slowloris.
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return srv, hub, cfg
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fatal("server error", err)
	}
}

// run starts the server and blocks until it stops, returning any non-graceful
// error. It is separate from main so its deferred cleanup runs.
func run() error {
	srv, hub, cfg := initServer()

	// Graceful shutdown on SIGINT/SIGTERM (e.g. `docker stop`): stop accepting
	// new connections, drain in-flight requests, then tear down websockets.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Prune persisted messages past the retention window, daily.
	if hub.store != nil {
		go runChatCleanup(ctx, hub.store, chatRetentionDaysOf(cfg))
	}

	// TLS source: ACME-managed (certmagic) when enabled, else the self-signed
	// cert files. A non-nil tlsConfig means ACME is in play.
	var tlsConfig *tls.Config
	if cfg.HTTPS && cfg.ACME {
		var err error
		tlsConfig, err = acmeTLSConfig(cfg)
		if err != nil {
			fatal("error setting up ACME", err)
		}
		srv.TLSConfig = tlsConfig
	}

	// HTTP/3 (QUIC) over UDP alongside the TCP h1/h2 listener, sharing the cert
	// and handler. Clients learn about it from the Alt-Svc header and upgrade.
	var h3 *http3.Server
	if cfg.HTTPS {
		h3 = &http3.Server{Addr: srv.Addr, Handler: srv.Handler, TLSConfig: tlsConfig}
		go func() {
			var err error
			if tlsConfig != nil {
				err = h3.ListenAndServe()
			} else {
				err = h3.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
			}
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("http3 server error", "err", err)
			}
		}()
	}

	shutdownErr := make(chan error, 1)
	go func() {
		<-ctx.Done()
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		if h3 != nil {
			_ = h3.Close()
		}
		hub.stop() // close active websocket connections after the HTTP drain
		if hub.store != nil {
			_ = hub.store.close()
		}
		shutdownErr <- err
	}()

	log.Info("starting server", "addr", srv.Addr, "https", cfg.HTTPS, "acme", cfg.ACME)
	var err error
	switch {
	case tlsConfig != nil:
		// Cert comes from srv.TLSConfig (certmagic), so no files are passed.
		err = srv.ListenAndServeTLS("", "")
	case cfg.HTTPS:
		err = srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
	default:
		err = srv.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	// Block until graceful shutdown has finished draining before exiting.
	if sErr := <-shutdownErr; sErr != nil {
		return fmt.Errorf("graceful shutdown: %w", sErr)
	}
	log.Info("server stopped")
	return nil
}
