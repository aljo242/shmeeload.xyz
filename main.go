package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aljo242/web_serve/handlers"
	"github.com/aljo242/web_serve/ip_util"

	"github.com/gorilla/mux"
)

const (
	DefaultPort = "80"

	DefaultHost = "localhost"
)

var (
	// Port of the HTTP Server
	Port = "80"
)

func startServer(wg *sync.WaitGroup) *http.Server {
	h, err := ip_util.HostInfo()
	if err != nil {
		log.Fatalf("Error creating Host Struct : %v", err)
		return nil
	}

	hostIP, err := ip_util.SelectHost(h.InternalIPs)
	if err != nil {
		log.Fatalf("Error chosing host IP : %v", err)
		return nil
	}

	addr := hostIP + ":" + Port
	// create new gorilla mux router
	r := mux.NewRouter()
	// attach pather with handler
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", handlers.ArticleHandler).Name("articleRoute")
	r.HandleFunc("/home", handlers.HomeHandler)
	r.HandleFunc("/", handlers.RedirectHome)
	r.HandleFunc("/scripts/{scriptname}", handlers.ScriptsHandler("bob"))
	r.HandleFunc("/css/{filename}", handlers.CSSHandler("Joe"))

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
