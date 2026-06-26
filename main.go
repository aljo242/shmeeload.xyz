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
	"github.com/gorilla/mux"
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
func buildRouter(cfg Config, hub *Hub) *mux.Router {
	r := mux.NewRouter()
	r.Use(securityHeaders)

	site := siteFS()

	page := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, rq *http.Request) {
			if rq.Method != http.MethodGet && rq.Method != http.MethodHead {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			http.ServeFileFS(w, rq, site, name)
		}
	}
	underConstruction := func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, "/under-construction", http.StatusTemporaryRedirect)
	}

	// dynamic endpoints
	r.HandleFunc("/donate/{cryptoname}", handlers.DonateHandler(cfg.CacheMaxAge))
	r.HandleFunc("/chat/ws", serveWs(hub))

	// pages (pretty URL -> embedded HTML)
	r.HandleFunc("/", func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, "/home", http.StatusPermanentRedirect)
	})
	r.HandleFunc("/home", page("home.html"))
	r.HandleFunc("/resume/home", page("resume.html"))
	r.HandleFunc("/hall-of-art/home", page("shadow.html"))
	r.HandleFunc("/chat/home", page("chat.html"))
	r.HandleFunc("/under-construction", page("construction.html"))

	// not-yet-built sections
	r.HandleFunc("/tunes/home", underConstruction)
	r.HandleFunc("/shop/home", underConstruction)
	r.HandleFunc("/chat/signup", underConstruction)
	r.HandleFunc("/chat/signin", underConstruction)

	// everything else is a static asset from the embedded site
	r.PathPrefix("/").Handler(http.FileServerFS(site))

	return r
}

// initServer loads config and returns a configured (but not yet listening)
// http.Server with the chat hub running.
func initServer() (*http.Server, *Hub, Config) {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		fatal("error loading config", err)
	}
	log.Setup(cfg.DebugLog)

	hub := newHub()
	go hub.run()

	srv := &http.Server{
		Addr:              cfg.IP + ":" + cfg.Port,
		Handler:           buildRouter(cfg, hub),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
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

	shutdownErr := make(chan error, 1)
	go func() {
		<-ctx.Done()
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
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
