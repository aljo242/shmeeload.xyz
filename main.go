package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	//"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/aljo242/ip_util"
	"github.com/aljo242/shmeeload.xyz/handlers"

	"github.com/gorilla/mux"
)

const (
	// ConfigFile is the name of the user's config JSON file
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
	DebugLogln(cfg.DebugLog, "setting up templates...")

	DebugLogln(cfg.DebugLog, "cleaning output directory...")
	// clean static output dir
	err := os.RemoveAll(TemplateOutputDir)
	if err != nil {
		return nil,
			fmt.Errorf("error cleaning ouput directory %v : %w", TemplateOutputDir, err)
	}

	DebugLogln(cfg.DebugLog, "creating new output directories...")
	// Create/ensure output directory
	if !Exists(TemplateOutputDir) {
		err := os.Mkdir(TemplateOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", TemplateOutputDir, err)
		}
	}

	// Create subdirs
	htmlOutputDir := filepath.Join(TemplateOutputDir, "html")
	if !Exists(htmlOutputDir) {
		err := os.Mkdir(htmlOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", htmlOutputDir, err)
		}
	}
	jsOutputDir := filepath.Join(TemplateOutputDir, "js")
	if !Exists(jsOutputDir) {
		err := os.Mkdir(jsOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", jsOutputDir, err)
		}
	}

	cssOutputDir := filepath.Join(TemplateOutputDir, "css")
	if !Exists(cssOutputDir) {
		err := os.Mkdir(cssOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", cssOutputDir, err)
		}
	}

	tsOutputDir := filepath.Join(TemplateOutputDir, "src")
	if !Exists(tsOutputDir) {
		err := os.Mkdir(tsOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("error creating directory %v : %w", tsOutputDir, err)
		}
	}

	DebugLogln(cfg.DebugLog, "ensuring template base directory exists...")
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

			switch filepath.Ext(path) {
			case ".html":
				newPath := filepath.Join(TemplateOutputDir, "html", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, "\t"+path+" -> "+newPath)
				ExecuteTemplateHTML(cfg, path, newPath)
			case ".js":
				newPath := filepath.Join(TemplateOutputDir, "js", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, "\t"+path+" -> "+newPath)
				CopyFile(path, newPath)
			case ".map":
				newPath := filepath.Join(TemplateOutputDir, "js", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, "\t"+path+" -> "+newPath)
				CopyFile(path, newPath)
			case ".css":
				newPath := filepath.Join(TemplateOutputDir, "css", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, "\t"+path+" -> "+newPath)
				CopyFile(path, newPath)
			case ".ts":
				newPath := filepath.Join(TemplateOutputDir, "src", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, "\t"+path+" -> "+newPath)
				CopyFile(path, newPath)
			}

			return nil
		})
	if err != nil {
		return nil,
			fmt.Errorf("error walking %v : %w", TemplateBaseDir, err)
	}

	DebugLogln(cfg.DebugLog, "template setup complete.")
	return files, nil
}

func getTLSConfig(cfg ServerConfig) (*tls.Config, error) {
	cer, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return &tls.Config{}, fmt.Errorf("error loading key pair (%v, %v) : %w", cfg.CertFile, cfg.KeyFile, err)
	}

	rootCAPool := x509.NewCertPool()

	// read rootCA file into byte
	f, err := os.Open(cfg.RootCA)
	if err != nil {
		return &tls.Config{}, fmt.Errorf("error opening Root CA file %v : %w", cfg.RootCA, err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return &tls.Config{}, fmt.Errorf("error reading Root CA file %v : %w", cfg.RootCA, err)
	}

	ok := rootCAPool.AppendCertsFromPEM(b)
	if !ok {
		return &tls.Config{}, fmt.Errorf("error appending Root CA cert %v : %w", cfg.RootCA, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cer},
		RootCAs:      rootCAPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func initServer(wg *sync.WaitGroup) (*Server, *ServerConfig) {
	cfg, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("error loading config : %v", err)
		return nil, nil
	}

	cfg.Print()

	var hostIP string
	if cfg.ChooseIP {

		h, err := ip_util.HostInfo()
		if err != nil {
			log.Fatalf("error creating Host Struct : %v", err)
			return nil, nil
		}

		hostIP, err = ip_util.SelectHost(h.InternalIPs)
		if err != nil {
			log.Fatalf("error chosing host IP : %v", err)
			return nil, nil
		}
	} else {
		hostIP = cfg.IP
	}

	_, err = SetupTemplates(cfg)
	if err != nil {
		log.Fatalf("error setting up templates: %v", err)
		return nil, nil
	}

	hub := newHub()
	go hub.run()

	addr := hostIP + ":" + cfg.Port

	// generate/execute resource templates

	// create new gorilla mux router
	r := mux.NewRouter()
	// attach pather with handler
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", handlers.ArticleHandler).Name("articleRoute")
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

	go func() {
		defer wg.Done() // let main know we are done cleaning up
		// always returns error. ErrServerClosed on graceful close
		if cfg.HTTPS {
			// listen for HTTP traffic and redirect to HTTPS
			go func(hostName string) {
				httpAddr := hostIP + ":80"
				httpsHost := "https://" + hostName
				log.Printf("redirecting all traffic to http://%v/* to %v/*", httpAddr, httpsHost)
				if err := http.ListenAndServe(httpAddr, http.HandlerFunc(handlers.RedirectHTTPS(httpsHost, cfg.DebugLog))); err != nil {
					log.Fatalf("ListenAndServe error: %v", err)
				}
			}(cfg.Host)

			if err = srv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
				// unexpected error
				log.Fatalf("ListenAndServeTLS() NOT IMPLEMENTED: %v", err)
			}
		} else {
			if err = srv.ListenAndServe(); err != http.ErrServerClosed {
				// unexpected error
				log.Fatalf("ListenAndServe(): %v", err)
			}
		}
	}()

	// return reference so caller can call Shutdown
	return srv, &cfg
}

func initServer2() *Server {
	log.Printf("loading configuration in file: %v", configFile)
	cfg, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("error loading config : %v", err)
		return nil
	}

	cfg.Print()

	var hostIP string
	if cfg.ChooseIP {

		h, err := ip_util.HostInfo()
		if err != nil {
			log.Fatalf("error creating Host Struct : %v", err)
			return nil
		}

		hostIP, err = ip_util.SelectHost(h.InternalIPs)
		if err != nil {
			log.Fatalf("error chosing host IP : %v", err)
			return nil
		}
	} else {
		hostIP = cfg.IP
	}

	_, err = SetupTemplates(cfg)
	if err != nil {
		log.Fatalf("error setting up templates: %v", err)
		return nil
	}

	hub := newHub()
	go hub.run()

	addr := hostIP + ":" + cfg.Port

	// generate/execute resource templates

	// create new gorilla mux router
	r := mux.NewRouter()
	// attach pather with handler
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", handlers.ArticleHandler).Name("articleRoute")
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

func runGorillaServer() {
	log.Printf("main: starting HTTP server...")

	httpServerExitDone := &sync.WaitGroup{}

	httpServerExitDone.Add(1)
	srv, cfg := initServer(httpServerExitDone)

	shutdownCh := make(chan interface{})
	getUserInput := func(ch chan<- interface{}) {
		var code int
		for {
			fmt.Printf("provide shutdown code: \n")
			fmt.Scanln(&code)
			if code == cfg.ShutdownCode {
				break
			}

			fmt.Printf("invalid code.\n")
		}
		ch <- code
	}

	go getUserInput(shutdownCh)
	select {
	case <-shutdownCh:
		if err := srv.Shutdown(context.Background()); err != nil {
			panic(err)
		}
		break
	}

	// wait for goroutine to stop
	httpServerExitDone.Wait()

	log.Printf("main: done. exiting...")
}

func main() {
	flag.Parse()
	//runGorillaServer()
	log.Printf("main: starting HTTP server...")
	srv := initServer2()
	srv.Run()
}
