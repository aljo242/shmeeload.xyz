package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aljo242/shmeeload.xyz/handlers"
	"github.com/aljo242/shmeeload.xyz/internal/log"
	"github.com/quic-go/quic-go/http3"
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

var configFile string

func init() {
	flag.StringVar(&configFile, "c", DefaultConfigFile, "Full path to JSON configuration file")
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

	// serveAsset serves a named embedded asset, 404ing if absent.
	serveAsset := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, rq *http.Request) {
			if !site.serve(w, rq, name) {
				http.NotFound(w, rq)
			}
		}
	}
	page := serveAsset
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
	mux.HandleFunc("GET /under-construction", page("construction.html"))

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
		if n, err := store.purgeOlderThan(window); err != nil {
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
	log.Setup(cfg.DebugLog)

	// With ACME on, certmagic manages the certificate; otherwise generate a
	// self-signed one for LAN use.
	if cfg.HTTPS && !cfg.ACME {
		if err := ensureCert(cfg.CertFile, cfg.KeyFile, cfg.TLSHosts); err != nil {
			fatal("error preparing TLS certificate", err)
		}
	}

	site, err := newStaticSite(siteFS(), cfg.CacheMaxAge)
	if err != nil {
		fatal("error indexing embedded site", err)
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
			if err != nil && err != http.ErrServerClosed {
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
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	// Block until graceful shutdown has finished draining before exiting.
	if sErr := <-shutdownErr; sErr != nil {
		return fmt.Errorf("graceful shutdown: %w", sErr)
	}
	log.Info("server stopped")
	return nil
}
