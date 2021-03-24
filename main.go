package main

import (
	"flag"
	"fmt"

	//"net"
	"os"
	"path/filepath"

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
)

var configFile string

func init() {
	flag.StringVar(&configFile, "c", DefaultConfigFile, "Full path to JSON configuration file")
}

// SetupTemplates builds the template output directory, executes HTML templates,
// and copies all web resource files to the template output directory (.js, .ts, .js.map, .css, .html)
func SetupTemplates(cfg ServerConfig) ([]string, error) {
	files := make([]string, 0)
	log.Debug().Msg("setting up templates")

	log.Debug().Msg("cleaning output directory")
	// clean static output dir
	err := os.RemoveAll(TemplateOutputDir)
	if err != nil {
		return nil,
			fmt.Errorf("error cleaning ouput directory %v : %w", TemplateOutputDir, err)
	}

	log.Debug().Str("OutputDir", TemplateOutputDir).Msg("creating new output directories")
	// Create/ensure output directory
	if !Exists(TemplateOutputDir) {
		err := os.Mkdir(TemplateOutputDir, 0750)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", TemplateOutputDir, err)
		}
	}

	// Create subdirs
	htmlOutputDir := filepath.Join(TemplateOutputDir, "html")
	if !Exists(htmlOutputDir) {
		err := os.Mkdir(htmlOutputDir, 0750)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", htmlOutputDir, err)
		}
	}
	jsOutputDir := filepath.Join(TemplateOutputDir, "js")
	if !Exists(jsOutputDir) {
		err := os.Mkdir(jsOutputDir, 0750)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", jsOutputDir, err)
		}
	}

	cssOutputDir := filepath.Join(TemplateOutputDir, "css")
	if !Exists(cssOutputDir) {
		err := os.Mkdir(cssOutputDir, 0750)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", cssOutputDir, err)
		}
	}

	tsOutputDir := filepath.Join(TemplateOutputDir, "src")
	if !Exists(tsOutputDir) {
		err := os.Mkdir(tsOutputDir, 0750)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", tsOutputDir, err)
		}
	}

	log.Debug().Str("BaseDir", TemplateBaseDir).Msg("ensuring template base directory exists")
	// Ensure base template directory exists
	if !Exists(TemplateBaseDir) {
		return nil,
			fmt.Errorf("base Dir %v does not exist", TemplateBaseDir)
	}

	// walk through all files in the template resource dir
	err = filepath.Walk(TemplateBaseDir,
		func(path string, info os.FileInfo, err error) error {
			// skip certain directories
			if info.IsDir() && info.Name() == "node_modules" {
				return filepath.SkipDir
			}

			handleCopyFileErr := func(err error) {
				if err != nil {
					log.Fatal().Err(err).Msg("error copying file")
				}
			}

			handleExecuteTemlateErr := func(err error) {
				if err != nil {
					log.Fatal().Err(err).Msg("error executing HTML template")
				}
			}

			switch filepath.Ext(path) {
			case ".html":
				newPath := filepath.Join(TemplateOutputDir, "html", filepath.Base(path))
				log.Debug().Str("fromPath", path).Str("toPath", newPath).Msg("moving static web resources")
				handleExecuteTemlateErr(ExecuteTemplateHTML(cfg, path, newPath))
			case ".js":
				newPath := filepath.Join(TemplateOutputDir, "js", filepath.Base(path))
				log.Debug().Str("fromPath", path).Str("toPath", newPath).Msg("moving static web resources")
				handleCopyFileErr(CopyFile(path, newPath))
			case ".map":
				newPath := filepath.Join(TemplateOutputDir, "js", filepath.Base(path))
				log.Debug().Str("fromPath", path).Str("toPath", newPath).Msg("moving static web resources")
				handleCopyFileErr(CopyFile(path, newPath))
			case ".css":
				newPath := filepath.Join(TemplateOutputDir, "css", filepath.Base(path))
				log.Debug().Str("fromPath", path).Str("toPath", newPath).Msg("moving static web resources")
				handleCopyFileErr(CopyFile(path, newPath))
			case ".ts":
				newPath := filepath.Join(TemplateOutputDir, "src", filepath.Base(path))
				log.Debug().Str("fromPath", path).Str("toPath", newPath).Msg("moving static web resources")
				handleCopyFileErr(CopyFile(path, newPath))
			}

			return nil
		})
	if err != nil {
		return nil,
			fmt.Errorf("error walking %v : %w", TemplateBaseDir, err)
	}

	log.Debug().Msg("template setup complete.")
	return files, nil
}

func initServer() *Server {
	log.Printf("loading configuration in file: %v", configFile)
	cfg, err := loadConfig(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading config")
		return nil
	}
	setupLogger(cfg)

	cfg.Print()

	var hostIP string
	if cfg.ChooseIP {

		h, err := ip_util.HostInfo()
		if err != nil {
			log.Fatal().Err(err).Msg("error creating Host Struct")
			return nil
		}

		hostIP, err = ip_util.SelectHost(h.InternalIPs)
		if err != nil {
			log.Fatal().Err(err).Msg("error chosing host IP")
			return nil
		}
	} else {
		hostIP = cfg.IP
	}

	_, err = SetupTemplates(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error setting up templates")
		return nil
	}

	hub := newHub()
	go hub.run()

	addr := hostIP + ":" + cfg.Port

	// generate/execute resource templates

	// create new gorilla mux router
	r := mux.NewRouter()
	// attach pather with handler
	r.HandleFunc("/home", handlers.HomeHandler)
	r.HandleFunc("/", handlers.RedirectHome(cfg.Host, cfg.DebugLog))
	r.HandleFunc("/static/js/{scriptname}", handlers.ScriptsHandler("bob", cfg.DebugLog))
	r.HandleFunc("/static/css/{filename}", handlers.CSSHandler("Joe", cfg.DebugLog))
	r.HandleFunc("/static/html/{filename}", handlers.HTMLHandler("Joe", cfg.DebugLog))
	r.HandleFunc("/static/src/{filename}", handlers.TypeScriptHandler("", cfg.DebugLog))
	r.HandleFunc("/chat/home", handlers.ChatHomeHandler("", cfg.DebugLog))
	//r.HandleFunc("/chat/{name}", handlers.ChatHomeHandler("", cfg.DebugLog))
	r.HandleFunc("/chat/ws", serveWs(hub))
	r.HandleFunc("/resume/home", handlers.ResumeHomeHandler(cfg.DebugLog))

	fmt.Printf("\n")
	log.Printf("starting Server at: %v...", addr)
	srv := NewServer(cfg, r)

	return srv
}

func main() {
	flag.Parse()
	//runGorillaServer()
	log.Printf("main: starting HTTP server...")
	srv := initServer()
	srv.Run()
}
