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

	"github.com/aljo242/ip_util"
	"github.com/aljo242/shmeeload.xyz/handlers"
	"github.com/aljo242/shmeeload.xyz/internal/log"
	"github.com/gorilla/mux"
)

const (
	// DefaultConfigFile is the default path to the JSON configuration file.
	DefaultConfigFile string = "sample/sample_config.json"

	// WebResourceDir holds the HTML/CSS/TS/image sources built into the in-memory
	// asset map at startup.
	WebResourceDir string = "./web_res"

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

// buildRouter wires every route to its handler and installs shared middleware.
func buildRouter(cfg Config, hub *Hub) *mux.Router {
	r := mux.NewRouter()
	r.Use(securityHeaders)

	r.HandleFunc("/home", handlers.HomeHandler(cfg.CacheMaxAge))
	r.HandleFunc("/", handlers.RedirectHome())
	r.HandleFunc("/static/js/{scriptname}", handlers.ScriptsHandler(cfg.CacheMaxAge))
	r.HandleFunc("/static/css/{filename}", handlers.CSSHandler(cfg.CacheMaxAge))
	r.HandleFunc("/static/html/{filename}", handlers.HTMLHandler(cfg.CacheMaxAge))
	r.HandleFunc("/static/src/{filename}", handlers.TypeScriptHandler(cfg.CacheMaxAge))
	r.HandleFunc("/static/img/{filename}", handlers.ImageHandler(cfg.CacheMaxAge))
	r.HandleFunc("/static/model/{filename}", handlers.ModelHandler(cfg.CacheMaxAge))
	r.HandleFunc("/manifest.json", handlers.ManifestHandler(cfg.CacheMaxAge))
	r.HandleFunc("/serviceWorker.js", handlers.ServiceWorkerHandler(cfg.CacheMaxAge))
	r.HandleFunc("/serviceWorker.js.map", handlers.ServiceWorkerHandler(cfg.CacheMaxAge))
	r.HandleFunc("/tunes/home", handlers.RedirectConstructionHandler())
	r.HandleFunc("/shop/home", handlers.RedirectConstructionHandler())

	// chat
	r.HandleFunc("/chat/home", handlers.ChatHomeHandler(cfg.CacheMaxAge))
	r.HandleFunc("/chat/ws", serveWs(hub))
	r.HandleFunc("/chat/signup", handlers.RedirectConstructionHandler())
	r.HandleFunc("/chat/signin", handlers.RedirectConstructionHandler())

	r.HandleFunc("/files/{filename}", handlers.MiscFileHandler(cfg.CacheMaxAge))
	r.HandleFunc("/resume/home", handlers.ResumeHomeHandler(cfg.CacheMaxAge))
	r.HandleFunc("/under-construction", handlers.ConstructionHandler(cfg.CacheMaxAge))
	r.HandleFunc("/hall-of-art/home", handlers.HallofArtHomeHandler(cfg.CacheMaxAge))
	r.HandleFunc("/donate/{cryptoname}", handlers.DonateHandler(cfg.CacheMaxAge))

	return r
}

// initServer loads config, builds the in-memory assets, and returns a configured
// (but not yet listening) http.Server along with the loaded config.
func initServer() (*http.Server, *Hub, Config) {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		fatal("error loading config", err)
	}
	log.Setup(cfg.DebugLog)

	hostIP := cfg.IP
	if cfg.ChooseIP {
		h, err := ip_util.HostInfo()
		if err != nil {
			fatal("error gathering host info", err)
		}
		hostIP, err = ip_util.SelectHost(h.InternalIPs)
		if err != nil {
			fatal("error choosing host IP", err)
		}
	}

	assets, err := buildAssets(WebResourceDir)
	if err != nil {
		fatal("error building assets", err)
	}
	handlers.SetAssets(assets, time.Now())

	hub := newHub()
	go hub.run()

	srv := &http.Server{
		Addr:              hostIP + ":" + cfg.Port,
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
