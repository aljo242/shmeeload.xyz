package main

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// mcHeadProxy serves Minecraft avatar heads same-origin so the gamers page can
// show who is online without loosening its strict img-src CSP and without
// leaking every visitor's IP to a third-party avatar host. It fetches each head
// once from the upstream service, then serves it from an in-memory cache.
type mcHeadProxy struct {
	client  *http.Client
	baseURL string // upstream avatar base, e.g. "https://mc-heads.net/avatar/"
	size    string // requested head size in pixels

	mu    sync.Mutex
	cache map[string]headEntry
}

type headEntry struct {
	png []byte // nil means the upstream had no head (cached negative)
	exp time.Time
}

const (
	headTTL      = 10 * time.Minute
	headMaxBytes = 128 << 10 // cap the upstream read; a 40px head is a few KB
	headMaxCache = 256       // bound memory; whole crew fits many times over
)

// headIDRe accepts a Minecraft username or a UUID (with or without dashes). It
// also fixes the upstream URL host, so the fetch below cannot be pointed
// elsewhere.
var headIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,36}$`)

func newMCHeadProxy() *mcHeadProxy {
	return &mcHeadProxy{
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: "https://mc-heads.net/avatar/",
		size:    "40",
		cache:   make(map[string]headEntry),
	}
}

// get returns the PNG head for id (username or UUID). The bool is false when the
// id is malformed or no head is available, in which case the caller should 404
// and let the page hide the broken avatar.
func (p *mcHeadProxy) get(ctx context.Context, id string) ([]byte, bool) {
	if !headIDRe.MatchString(id) {
		return nil, false
	}
	now := time.Now()

	p.mu.Lock()
	if e, ok := p.cache[id]; ok && now.Before(e.exp) {
		png := e.png
		p.mu.Unlock()
		return png, png != nil
	}
	p.mu.Unlock()

	png := p.fetch(ctx, id)

	p.mu.Lock()
	if len(p.cache) >= headMaxCache {
		p.cache = make(map[string]headEntry, headMaxCache) // simplest bound: drop all when full
	}
	p.cache[id] = headEntry{png: png, exp: now.Add(headTTL)}
	p.mu.Unlock()

	return png, png != nil
}

// fetch pulls a single head from the upstream service. Any failure returns nil,
// which is cached as a short-lived negative so a flaky upstream is not hammered.
func (p *mcHeadProxy) fetch(ctx context.Context, id string) []byte {
	url := p.baseURL + id + "/" + p.size + ".png"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, headMaxBytes))
	if err != nil {
		return nil
	}
	return b
}
