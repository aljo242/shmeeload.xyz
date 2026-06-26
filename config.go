package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the server configuration loaded from a JSON file.
type Config struct {
	Port        string `json:"port"`
	IP          string `json:"IP"`
	ChooseIP    bool   `json:"chooseIP"`
	HTTPS       bool   `json:"secure"`
	DebugLog    bool   `json:"debugLog"`
	CacheMaxAge int    `json:"cacheMaxAge"`
	CertFile    string `json:"certFile"`
	KeyFile     string `json:"keyFile"`
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
