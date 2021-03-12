package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aljo242/ip_util"
	"github.com/aljo242/shmeeload.xyz/handlers"

	"github.com/gorilla/mux"
)

const (
	// ConfigFile is the name of the user's config JSON file
	ConfigFile string = "config.json"

	// TemplateBaseDir is where HTML template files are located to be
	// executed and copied to the res dir
	TemplateBaseDir string = "./web_res"

	// TemplateOutputDir is the directory all outputs of SetupTemplates will fall under
	TemplateOutputDir string = "./static"
)

// Exists is a basic file util that says if a dir or file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return true // path/file exists
	}
	return false // path/file does not exist
}

// CopyFile copies filename src to dst
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Error opening file: %v : %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Error creating file: %v : %w", src, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("Error copying %v to %v : %w", src, dst, err)
	}

	return nil
}

// SetupTemplates builds the template output directory, executes HTML templates,
// and copies all web resource files to the template output directory (.js, .ts, .js.map, .css, .html)
func SetupTemplates(cfg ServerConfig) ([]string, error) {
	files := make([]string, 0)
	DebugLogln(cfg.DebugLog, "SETTING UP TEMPLATES")

	DebugLogln(cfg.DebugLog, "Cleaning output directory...")
	// clean static output dir
	err := os.RemoveAll(TemplateOutputDir)
	if err != nil {
		return nil,
			fmt.Errorf("Error cleaning ouput directory %v : %w", TemplateOutputDir, err)
	}

	DebugLogln(cfg.DebugLog, "Creating new output directories...")
	// Create/ensure output directory
	if !Exists(TemplateOutputDir) {
		err := os.Mkdir(TemplateOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("Error creating directory %v : %w", TemplateOutputDir, err)
		}
	}

	// Create subdirs
	htmlOutputDir := filepath.Join(TemplateOutputDir, "html")
	if !Exists(htmlOutputDir) {
		err := os.Mkdir(htmlOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("Error creating directory %v : %w", htmlOutputDir, err)
		}
	}
	jsOutputDir := filepath.Join(TemplateOutputDir, "js")
	if !Exists(jsOutputDir) {
		err := os.Mkdir(jsOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("Error creating directory %v : %w", jsOutputDir, err)
		}
	}

	cssOutputDir := filepath.Join(TemplateOutputDir, "css")
	if !Exists(cssOutputDir) {
		err := os.Mkdir(cssOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("Error creating directory %v : %w", cssOutputDir, err)
		}
	}

	tsOutputDir := filepath.Join(TemplateOutputDir, "src")
	if !Exists(tsOutputDir) {
		err := os.Mkdir(tsOutputDir, 0755)
		if err != nil {
			return nil,
				fmt.Errorf("Error creating directory %v : %w", tsOutputDir, err)
		}
	}

	DebugLogln(cfg.DebugLog, "Ensuring template base directory exists...")
	// Ensure base template directory exists
	if !Exists(TemplateBaseDir) {
		return nil,
			fmt.Errorf("Base Dir %v does not exist", TemplateBaseDir)
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
				DebugPrintln(cfg.DebugLog, path+" -> "+newPath)
				ExecuteTemplateHTML(cfg, path, newPath)
			case ".js":
				newPath := filepath.Join(TemplateOutputDir, "js", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, path+" -> "+newPath)
				CopyFile(path, newPath)
			case ".map":
				newPath := filepath.Join(TemplateOutputDir, "js", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, path+" -> "+newPath)
				CopyFile(path, newPath)
			case ".css":
				newPath := filepath.Join(TemplateOutputDir, "css", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, path+" -> "+newPath)
				CopyFile(path, newPath)
			case ".ts":
				newPath := filepath.Join(TemplateOutputDir, "src", filepath.Base(path))
				DebugPrintln(cfg.DebugLog, path+" -> "+newPath)
				CopyFile(path, newPath)
			}

			return nil
		})
	if err != nil {
		return nil,
			fmt.Errorf("Error walking %v : %w", TemplateBaseDir, err)
	}

	return files, nil
}

func getTLSConfig(cfg ServerConfig) (*tls.Config, error) {
	cer, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return &tls.Config{}, fmt.Errorf("Error Loading Key Pair (%v, %v) : %w", cfg.CertFile, cfg.KeyFile, err)
	}

	rootCAPool := x509.NewCertPool()

	// read rootCA file into byte
	f, err := os.Open(cfg.RootCA)
	if err != nil {
		return &tls.Config{}, fmt.Errorf("Error Opening Root CA file %v : %w", cfg.RootCA, err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return &tls.Config{}, fmt.Errorf("Error Reading Root CA file %v : %w", cfg.RootCA, err)
	}

	ok := rootCAPool.AppendCertsFromPEM(b)
	if !ok {
		return &tls.Config{}, fmt.Errorf("Error appending root CA cert %v : %w", cfg.RootCA, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cer},
		RootCAs:      rootCAPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func startServer(wg *sync.WaitGroup) (*http.Server, *ServerConfig) {
	cfg, err := loadConfig(ConfigFile)
	if err != nil {
		log.Fatalf("Error loading config : %v", err)
		return nil, nil
	}

	cfg.Print()

	var hostIP string
	if cfg.ChooseIP {

		h, err := ip_util.HostInfo()
		if err != nil {
			log.Fatalf("Error creating Host Struct : %v", err)
			return nil, nil
		}

		hostIP, err = ip_util.SelectHost(h.InternalIPs)
		if err != nil {
			log.Fatalf("Error chosing host IP : %v", err)
			return nil, nil
		}
	} else {
		hostIP = cfg.IP
	}

	_, err = SetupTemplates(cfg)
	if err != nil {
		log.Fatalf("Error setting up templates: %v", err)
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

	srv := &http.Server{
		Handler:        r,
		Addr:           addr,
		WriteTimeout:   15 * time.Second,
		ReadTimeout:    15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// add TLS Config if using HTTPS
	if cfg.HTTPS {
		// TODO FLESH OUT
		srv.TLSConfig, err = getTLSConfig(cfg)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Using HTTPS\n")
		log.Printf("Key Pair:\t(%v, %v)\n", cfg.CertFile, cfg.KeyFile)
		//log.Println(srv.TLSConfig)
	}

	log.Printf("Starting Server at: %v...", addr)
	go func() {
		defer wg.Done() // let main know we are done cleaning up
		// always returns error. ErrServerClosed on graceful close
		if cfg.HTTPS {
			// listen for HTTP traffic and redirect to HTTPS
			go func(hostName string) {
				httpAddr := hostIP + ":80"
				httpsHost := "https://" + hostName
				log.Printf("Redirecting all traffic to http://%v/* to %v/*", httpAddr, httpsHost)
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

func runGorillaServer() {
	log.Printf("main: starting HTTP server...")

	httpServerExitDone := &sync.WaitGroup{}

	httpServerExitDone.Add(1)
	srv, cfg := startServer(httpServerExitDone)

	shutdownCh := make(chan int)
	getUserInput := func(ch chan<- int) {
		var code int
		for {
			fmt.Printf("Provide shutdown code: \n")
			fmt.Scanln(&code)
			if code == cfg.ShutdownCode {
				break
			}

			fmt.Printf("Invalid Code.\n")
		}
		ch <- code
	}

	go getUserInput(shutdownCh)
	select {
	case code := <-shutdownCh:
		if err := srv.Shutdown(context.TODO()); err != nil {
			panic(err)
		}
		log.Printf("main: shutdown code %d", code)
		break
	}

	// wait for goroutine to stop
	httpServerExitDone.Wait()

	log.Printf("main: done. exiting...")
}

func main() {
	runGorillaServer()
}
