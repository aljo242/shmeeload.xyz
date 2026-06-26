package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aljo242/ip_util"
	"github.com/aljo242/shmeeload.xyz/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

const (
	// DefaultConfigFile is the name of the user's config JSON file
	DefaultConfigFile string = "sample/sample_config.json"

	// TemplateBaseDir is where HTML template files are located to be
	// executed and copied to the res dir
	TemplateBaseDir string = "./web_res"

	// TemplateOutputDir is the directory all outputs of SetupTemplates will fall under
	TemplateOutputDir string = "./static"

	// shutdownTimeout bounds how long graceful shutdown waits for in-flight requests.
	shutdownTimeout = 10 * time.Second
)

var configFile string

func init() {
	flag.StringVar(&configFile, "c", DefaultConfigFile, "Full path to JSON configuration file")
}

// SetupTemplates rebuilds the static output directory: it cleans and recreates
// the per-type output subdirectories, renders the HTML templates, and copies the
// remaining web resources (.js, .map, .css, .ts, images, models, misc files)
// into place. Any failure is returned rather than terminating the process.
func SetupTemplates() error {
	log.Debug().Msg("setting up templates")

	log.Debug().Msg("cleaning output directory")
	if err := os.RemoveAll(TemplateOutputDir); err != nil {
		return fmt.Errorf("error cleaning output directory %v : %w", TemplateOutputDir, err)
	}

	htmlOutputDir := filepath.Join(TemplateOutputDir, "html")
	jsOutputDir := filepath.Join(TemplateOutputDir, "js")
	cssOutputDir := filepath.Join(TemplateOutputDir, "css")
	tsOutputDir := filepath.Join(TemplateOutputDir, "src")
	imgOutputDir := filepath.Join(TemplateOutputDir, "img")
	modelOutputDir := filepath.Join(TemplateOutputDir, "model")
	miscFilesOutputDir := filepath.Join(TemplateOutputDir, "files")

	log.Debug().Str("OutputDir", TemplateOutputDir).Msg("creating new output directories")
	for _, dir := range []string{
		TemplateOutputDir, htmlOutputDir, jsOutputDir, cssOutputDir,
		tsOutputDir, imgOutputDir, modelOutputDir, miscFilesOutputDir,
	} {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}

	log.Debug().Str("BaseDir", TemplateBaseDir).Msg("ensuring template base directory exists")
	if !Exists(TemplateBaseDir) {
		return fmt.Errorf("base Dir %v does not exist", TemplateBaseDir)
	}

	// Walk the resource dir, rendering HTML templates and copying everything else
	// into the matching output subdirectory by extension.
	err := filepath.Walk(TemplateBaseDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			var newPath string
			switch filepath.Ext(path) {
			case ".html":
				newPath = filepath.Join(htmlOutputDir, filepath.Base(path))
				log.Debug().Str("fromPath", path).Str("toPath", newPath).Msg("rendering template")
				if err := ExecuteTemplateHTML(path, newPath); err != nil {
					return fmt.Errorf("error rendering template %v : %w", path, err)
				}
				return nil
			case ".js", ".map":
				newPath = filepath.Join(jsOutputDir, filepath.Base(path))
				// The service worker must live at the site root to control the whole scope.
				if base := filepath.Base(path); base == "serviceWorker.js" || base == "serviceWorker.js.map" {
					newPath = filepath.Join(".", base)
				}
			case ".css":
				newPath = filepath.Join(cssOutputDir, filepath.Base(path))
			case ".ts":
				newPath = filepath.Join(tsOutputDir, filepath.Base(path))
			case ".ico", ".png", ".jpg", ".jpeg", ".svg", ".gif":
				newPath = filepath.Join(imgOutputDir, filepath.Base(path))
			case ".pdf", ".doc", ".docx", ".xml":
				newPath = filepath.Join(miscFilesOutputDir, filepath.Base(path))
			case ".dae", ".obj", ".gltf":
				newPath = filepath.Join(modelOutputDir, filepath.Base(path))
			default:
				return nil
			}

			log.Debug().Str("fromPath", path).Str("toPath", newPath).Msg("copying web resource")
			if err := CopyFile(path, newPath); err != nil {
				return fmt.Errorf("error copying %v : %w", path, err)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("error walking %v : %w", TemplateBaseDir, err)
	}

	log.Debug().Msg("template setup complete.")
	return nil
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

// initServer loads config, prepares the static assets, and returns a configured
// (but not yet listening) http.Server along with the loaded config.
func initServer() (*http.Server, Config) {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading config")
	}
	setupLogger(cfg)

	hostIP := cfg.IP
	if cfg.ChooseIP {
		h, err := ip_util.HostInfo()
		if err != nil {
			log.Fatal().Err(err).Msg("error gathering host info")
		}
		hostIP, err = ip_util.SelectHost(h.InternalIPs)
		if err != nil {
			log.Fatal().Err(err).Msg("error choosing host IP")
		}
	}

	if err := SetupTemplates(); err != nil {
		log.Fatal().Err(err).Msg("error setting up templates")
	}

	hub := newHub()
	go hub.run()

	srv := &http.Server{
		Addr:              hostIP + ":" + cfg.Port,
		Handler:           buildRouter(cfg, hub),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return srv, cfg
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}

// run starts the server and blocks until it stops, returning any non-graceful
// error. It is separate from main so its deferred cleanup runs (log.Fatal in
// main would skip defers).
func run() error {
	srv, cfg := initServer()

	// Graceful shutdown on SIGINT/SIGTERM (e.g. `docker stop`): stop accepting
	// new connections and let in-flight requests drain before exiting.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Info().Msg("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("graceful shutdown failed")
		}
	}()

	log.Info().Str("addr", srv.Addr).Bool("https", cfg.HTTPS).Msg("starting server")
	var err error
	if cfg.HTTPS {
		err = srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	log.Info().Msg("server stopped")
	return nil
}
