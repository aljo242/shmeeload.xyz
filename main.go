package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
	mux.HandleFunc("GET /donate/{cryptoname}", handlers.DonateHandler(cfg.CacheMaxAge))
	mux.HandleFunc("GET /chat/ws", serveWs(hub, conns))

	// pages (pretty URL -> embedded HTML)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, "/home", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("GET /home", page("home.html"))
	mux.HandleFunc("GET /resume/home", page("resume.html"))
	mux.HandleFunc("GET /hall-of-art/home", page("shadow.html"))
	mux.HandleFunc("GET /chat/home", page("chat.html"))
	mux.HandleFunc("GET /under-construction", page("construction.html"))

	// not-yet-built sections
	mux.HandleFunc("GET /tunes/home", underConstruction)
	mux.HandleFunc("GET /shop/home", underConstruction)
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
	h = securityHeaders(h)
	return h
}

// initServer loads config and returns a configured (but not yet listening)
// http.Server with the chat hub running.
func initServer() (*http.Server, *Hub, Config) {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		fatal("error loading config", err)
	}
	log.Setup(cfg.DebugLog)

	if cfg.HTTPS {
		if err := ensureCert(cfg.CertFile, cfg.KeyFile, cfg.TLSHosts); err != nil {
			fatal("error preparing TLS certificate", err)
		}
	}

	site, err := newStaticSite(siteFS(), cfg.CacheMaxAge)
	if err != nil {
		fatal("error indexing embedded site", err)
	}

	hub := newHub()
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

	// HTTP/3 (QUIC) over UDP alongside the TCP h1/h2 listener, sharing the cert
	// and handler. Clients learn about it from the Alt-Svc header and upgrade.
	var h3 *http3.Server
	if cfg.HTTPS {
		h3 = &http3.Server{Addr: srv.Addr, Handler: srv.Handler}
		go func() {
			if err := h3.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && err != http.ErrServerClosed {
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
		shutdownErr <- err
	}()

	log.Info("starting server", "addr", srv.Addr, "https", cfg.HTTPS)
	var err error
	if cfg.HTTPS {
		err = srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
	} else {
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
