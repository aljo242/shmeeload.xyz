package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aljo242/shmeeload.xyz/handlers"
	"github.com/gorilla/mux"
)

type Server struct {
	http.Server
	config ServerConfig
	wg     *sync.WaitGroup
}

func serverShutdownCallback() {
	fmt.Printf("\n")
	log.Printf("shutting down server...")
}

func NewServer(cfg ServerConfig, r *mux.Router) *Server {
	tlsCfg := &tls.Config{}
	addr := cfg.IP + ":" + cfg.Port

	if cfg.HTTPS {
		tlsCfg, _ = getTLSConfig(cfg)
		// TODO handle error
	}

	srv := &Server{
		http.Server{
			Handler:           r,
			Addr:              addr,
			WriteTimeout:      15 * time.Second,
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 15 * time.Second,
			MaxHeaderBytes:    1 << 20,
			TLSConfig:         tlsCfg,
		},
		cfg,
		&sync.WaitGroup{},
	}

	srv.RegisterOnShutdown(serverShutdownCallback)

	return srv
}

func (srv *Server) Run() {

	srv.wg.Add(1)

	go func() {
		defer srv.wg.Done() // let main know we are done cleaning up
		// always returns error. ErrServerClosed on graceful close
		if srv.config.HTTPS {
			// listen for HTTP traffic and redirect to HTTPS
			go func(hostName string) {
				httpAddr := srv.config.IP + ":80"
				httpsHost := "https://" + hostName
				log.Printf("redirecting all traffic to http://%v/* to %v/*", httpAddr, httpsHost)
				if err := http.ListenAndServe(httpAddr, http.HandlerFunc(handlers.RedirectHTTPS(httpsHost, srv.config.DebugLog))); err != nil {
					log.Fatalf("ListenAndServe error: %v", err)
				}
			}(srv.config.Host)

			if err := srv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
				// unexpected error
				log.Fatalf("ListenAndServeTLS() NOT IMPLEMENTED: %v", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				// unexpected error
				log.Fatalf("ListenAndServe(): %v", err)
			}
		}
	}()

	shutdownCh := make(chan interface{})
	getUserInput := func(ch chan<- interface{}) {
		var code int
		for {
			fmt.Printf("provide shutdown code: \n")
			fmt.Scanln(&code)
			if code == srv.config.ShutdownCode {
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
	srv.wg.Wait()

	log.Printf("main: done. exiting...")

}
