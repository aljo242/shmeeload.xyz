package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ServerConfig is the general struct holds parsed JSON config info
type ServerConfig struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	IP           string `json:"IP"`
	ChooseIP     bool   `json:"chooseIP"`
	HTTPS        bool   `json:"secure"`
	DebugLog     bool   `json:"debugLog"`
	ShutdownCode int    `json:"shutdownCode"`
	CertFile     string `json:"certFile"`
	KeyFile      string `json:"keyFile"`
	RootCA       string `json:"rootCA"`
	// TODO add more
}

func loadConfig(filename string) (ServerConfig, error) {
	cfg := ServerConfig{}
	cfgFile, err := os.Open(filename)
	defer cfgFile.Close()
	if err != nil {
		return ServerConfig{},
			fmt.Errorf("Error opening config file %v : %w", filename, err)
	}

	jsonParser := json.NewDecoder(cfgFile)
	err = jsonParser.Decode(&cfg)
	if err != nil {
		return ServerConfig{},
			fmt.Errorf("Error parsing file %v : %w", filename, err)
	}

	return cfg, nil
}
