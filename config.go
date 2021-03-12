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

// Print provides a pretty formatted print of a ServerConfig
func (cfg *ServerConfig) Print() {
	fmt.Printf("\n-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-\n")
	fmt.Printf("ServerConfig:\n")
	fmt.Printf("\tHost:\t\t%v\n", cfg.Host)
	fmt.Printf("\tPort:\t\t%v\n", cfg.Port)
	fmt.Printf("\tIP:\t\t%v\n", cfg.IP)
	fmt.Printf("\tChooseIP:\t%t\n", cfg.ChooseIP)
	fmt.Printf("\tHTTPS:\t\t%t\n", cfg.HTTPS)
	fmt.Printf("\tDebugLog:\t%t\n", cfg.DebugLog)
	fmt.Printf("\tDebugLog:\t%t\n", cfg.DebugLog)
	fmt.Printf("\tShutdownCode:\t%d\n", cfg.ShutdownCode)
	fmt.Printf("\tCertFile:\t%v\n", cfg.CertFile)
	fmt.Printf("\tKeyFile:\t%v\n", cfg.KeyFile)
	fmt.Printf("\tRootCA:\t\t%v\n", cfg.RootCA)
	fmt.Printf("-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-_-\n\n")
}
