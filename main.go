package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aljo242/shmeeload.xyz/handlers"
	"github.com/aljo242/shmeeload.xyz/ip_util"

	"github.com/gorilla/mux"
)

const (
	DefaultPort = "80"

	DefaultHost = "localhost"

	ConfigFile string = "config.json"
)

var (
	// Port of the HTTP Server
	Port = "80"
)

type config struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	IP       string `json:"IP"`
	ChooseIP bool   `json:"chooseIP"`
	HTTPS    bool   `json:"secure"`
	DebugLog bool   `json:"debugLog"`
	// TODO add more
}

func loadConfig(filename string) (config, error) {
	cfg := config{}
	cfgFile, err := os.Open(filename)
	defer cfgFile.Close()
	if err != nil {
		return config{},
			fmt.Errorf("Error opening config file %v : %w", filename, err)
	}

	jsonParser := json.NewDecoder(cfgFile)
	err = jsonParser.Decode(&cfg)
	if err != nil {
		return config{},
			fmt.Errorf("Error parsing file %v : %w", filename, err)
	}

	return cfg, nil
}

type htmlTemplateInfo struct {
	Host string
	// TODO add more
}

func setupTemplates() {

}

func startServer(wg *sync.WaitGroup) *http.Server {
	cfg, err := loadConfig(ConfigFile)
	if err != nil {
		log.Fatalf("Error loading config : %v", err)
		return nil
	}
	fmt.Printf("%v\n", cfg)

	var hostIP string
	if cfg.ChooseIP {

		h, err := ip_util.HostInfo()
		if err != nil {
			log.Fatalf("Error creating Host Struct : %v", err)
			return nil
		}

		hostIP, err = ip_util.SelectHost(h.InternalIPs)
		if err != nil {
			log.Fatalf("Error chosing host IP : %v", err)
			return nil
		}
	} else {
		hostIP = cfg.IP
	}

	addr := hostIP + ":" + cfg.Port

	// generate/execute resource templates

	// create new gorilla mux router
	r := mux.NewRouter()
	// attach pather with handler
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", handlers.ArticleHandler).Name("articleRoute")
	r.HandleFunc("/home", handlers.HomeHandler)
	r.HandleFunc("/", handlers.RedirectHome)
	r.HandleFunc("/scripts/{scriptname}", handlers.ScriptsHandler("bob", cfg.DebugLog))
	r.HandleFunc("/css/{filename}", handlers.CSSHandler("Joe", cfg.DebugLog))
	r.HandleFunc("/chat/home", handlers.ChatHomeHandler("", cfg.DebugLog))
	r.HandleFunc("/chat/{name}", handlers.ChatHomeHandler("", cfg.DebugLog))

	srv := &http.Server{
		Handler:      r,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Starting Server at: %v...", addr)
	go func() {
		defer wg.Done() // let main know we are done cleaning up
		// always returns error. ErrServerClosed on graceful close
		if err = srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// return reference so caller can call Shutdown
	return srv
}

const shutdownCode int = 10101

func runGorillaServer() {
	log.Printf("main: starting HTTP server...")

	httpServerExitDone := &sync.WaitGroup{}

	httpServerExitDone.Add(1)
	srv := startServer(httpServerExitDone)

	shutdownCh := make(chan int)
	getUserInput := func(ch chan<- int) {
		var code int
		for {
			fmt.Printf("Provide shutdown code: \n")
			fmt.Scanln(&code)
			if code == shutdownCode {
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
