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

	// ACME (Let's Encrypt) for a publicly-trusted cert. When ACME is true the
	// self-signed path is bypassed; CertFile/KeyFile are ignored.
	ACME        bool     `json:"acme"`
	ACMEStaging bool     `json:"acmeStaging"` // use the LE staging CA while testing
	ACMEEmail   string   `json:"acmeEmail"`   // contact address for the ACME account
	ACMEDir     string   `json:"acmeDir"`     // where managed certs are stored (a persistent dir)
	Domains     []string `json:"domains"`     // hostnames to obtain/serve certs for

	// Chat: a fixed set of rooms, message persistence, and retention-based cleanup.
	ChatRooms         []string `json:"chatRooms"`         // curated room names (defaults applied if empty)
	ChatDBPath        string   `json:"chatDBPath"`        // SQLite file for persisted messages (defaults to <acmeDir parent>/chat.db)
	ChatRetentionDays int      `json:"chatRetentionDays"` // delete messages older than this many days (default 14)

	TunesDir string `json:"tunesDir"` // directory of MP3s served on the tunes page (default /tunes)

	// MCPushToken is the shared bearer token every pusher presents: the game host
	// for POST /gamers/valheim, plus hostpush and edgelord for the internal
	// dashboard. The name predates Minecraft's retirement and is kept because
	// rotating it would mean touching four config files on two boxes at once.
	// Empty disables the ingest endpoints (GET still serves what has been pushed).
	MCPushToken string `json:"mcPushToken"`

	// Internal homelab view (Pi-hole stats etc.) at /internal, gated by HTTP Basic
	// Auth. All four must be set for the view to register; empty disables it and
	// none of this data ever appears on a public endpoint.
	InternalUser   string `json:"internalUser"`
	InternalPass   string `json:"internalPass"`
	PiholeURL      string `json:"piholeURL"`      // e.g. "http://192.168.68.56"
	PiholePassword string `json:"piholePassword"` // Pi-hole v6 API password

	// InternalTrustCIDRs are source networks (matched against Caddy's X-Real-IP)
	// that may reach /internal WITHOUT Basic Auth, e.g. the LAN and tailnet.
	// Empty means always require auth. Safe only because the app is loopback-only
	// and Caddy stamps X-Real-IP with the true peer, so it can't be forged.
	InternalTrustCIDRs []string `json:"internalTrustCIDRs"`

	// AlertWebhook receives a Discord message when the /internal roll-up status
	// changes, so a red banner does not wait for someone to open the page.
	// Empty disables alerting entirely.
	AlertWebhook string `json:"alertWebhook"`

	Dev bool `json:"dev"` // dev mode: serve site/ from disk (no embed/minify); also set by -dev
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
