package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aljo242/shmeeload.xyz/handlers"
	"github.com/gorilla/mux"

	"github.com/rs/zerolog/log"
)

type Server struct {
	http.Server
	config    ServerConfig
	wg        *sync.WaitGroup
	quit      chan struct{}
	isRunning bool
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

	quit := make(chan struct{})

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
		quit,
		false,
	}

	srv.RegisterOnShutdown(serverShutdownCallback)

	return srv
}

// Quit sends closes the server quit channel if the server is running
// signaling the server to begin shutting down
// if the server is not running, Quit will return an error
func (srv *Server) Quit() error {
	if srv.isRunning {
		close(srv.quit)
		srv.isRunning = !srv.isRunning
		return nil
	}

	return errors.New("server not running; cannot shutdown")
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
					log.Fatal().Err(err).Msg("ListenAndServe error")
				}
			}(srv.config.Host)

			if err := srv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
				// unexpected error
				log.Fatal().Err(err).Msg("ListenAndServeTLS() NOT IMPLEMENTED")
			}
		} else {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				// unexpected error
				log.Fatal().Err(err).Msg("ListenAndServe()")
			}
		}

	}()

	// once we have run ListenAdnServe*, we are officially running
	srv.isRunning = true

	getUserInput := func() {
		var code int
		for {
			fmt.Printf("provide shutdown code: \n")
			fmt.Scanln(&code)
			if code == srv.config.ShutdownCode {
				break
			}

			fmt.Printf("invalid code.\n")
		}

		//close(srv.quit)
		err := srv.Quit()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to quit server")
		}
	}

	go getUserInput()
	select {
	case <-srv.quit:
		if err := srv.Shutdown(context.Background()); err != nil {
			panic(err)
		}
		break
	}

	// wait for goroutine to stop
	srv.wg.Wait()

	log.Printf("main: done. exiting...")
}
