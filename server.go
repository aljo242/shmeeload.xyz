package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Server struct {
	http.Server
}

func serverShutdownCallback() {
	fmt.Printf("\n")
	log.Printf("shutting down server...")
}

func NewServer(cfg ServerConfig, r *mux.Router, addr string) *Server {
	tlsCfg := &tls.Config{}

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
	}

	srv.RegisterOnShutdown(serverShutdownCallback)

	return srv
}
