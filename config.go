package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the server configuration loaded from a JSON file.
type Config struct {
	Port        string   `json:"port"`
	IP          string   `json:"IP"`
	HTTPS       bool     `json:"secure"`
	DebugLog    bool     `json:"debugLog"`
	CacheMaxAge int      `json:"cacheMaxAge"`
	CertFile    string   `json:"certFile"`
	KeyFile     string   `json:"keyFile"`
	TLSHosts    []string `json:"tlsHosts"` // SANs for the self-signed cert generated when secure is true
	HSTS        bool     `json:"hsts"`     // send Strict-Transport-Security; enable only with a publicly-trusted cert
}

// LoadConfig reads and parses the JSON config at path. Unlike the previous
// chef.LoadConfig, it preserves the underlying cause (permission denied, parse
// error, etc.) instead of collapsing every failure into "file does not exist".
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config %q: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %q: %w", path, err)
	}
	return cfg, nil
}
