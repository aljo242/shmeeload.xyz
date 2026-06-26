package main

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Abuse-control limits. Generous enough that normal browsing (a page plus its
// assets, multiplexed over HTTP/2) never trips them, tight enough to stop a
// flood. The chat is an open broadcast room, so it gets the strictest caps.
const (
	httpRatePerSec = 50  // sustained HTTP requests per client IP
	httpBurst      = 100 // short burst (one page load fans out to many assets)
	wsMaxPerIP     = 5   // concurrent chat connections per client IP
	wsMaxTotal     = 200 // concurrent chat connections across all clients
	wsMsgPerSec    = 2   // chat messages per second per connection
	wsMsgBurst     = 5   // short message burst per connection

	ipEntryTTL = 10 * time.Minute // evict idle HTTP limiter entries after this
)

// clientIP returns the request's source IP. The binary terminates connections
// directly (no proxy), so RemoteAddr is the real client; if it is ever fronted,
// this is where a trusted X-Forwarded-For would be parsed instead.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ipRateLimiter throttles HTTP requests per client IP with a token bucket,
// sweeping idle entries so the map cannot grow without bound.
type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
	rate    rate.Limit
	burst   int
	stop    chan struct{}
}

type ipEntry struct {
	limiter *rate.Limiter
	seen    time.Time
}

func newIPRateLimiter(perSec rate.Limit, burst int) *ipRateLimiter {
	l := &ipRateLimiter{
		entries: make(map[string]*ipEntry),
		rate:    perSec,
		burst:   burst,
		stop:    make(chan struct{}),
	}
	go l.sweepLoop()
	return l
}

// Stop ends the background sweeper. The server lets it run for the process
// lifetime; tests call it so the goroutine exits.
func (l *ipRateLimiter) Stop() { close(l.stop) }

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	if !ok {
		e = &ipEntry{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.entries[ip] = e
	}
	e.seen = time.Now()
	return e.limiter.Allow()
}

func (l *ipRateLimiter) sweepLoop() {
	t := time.NewTicker(ipEntryTTL)
	defer t.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
			l.evictIdle(time.Now())
		}
	}
}

// evictIdle drops entries not seen within the TTL ending at now.
func (l *ipRateLimiter) evictIdle(now time.Time) {
	cutoff := now.Add(-ipEntryTTL)
	l.mu.Lock()
	defer l.mu.Unlock()
	for ip, e := range l.entries {
		if e.seen.Before(cutoff) {
			delete(l.entries, ip)
		}
	}
}

// middleware rejects requests from an IP that exceeds its rate with 429.
func (l *ipRateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// connLimiter caps concurrent websocket connections, both per IP and in total,
// so the chat hub cannot be exhausted by one client or by sheer numbers.
type connLimiter struct {
	mu       sync.Mutex
	perIP    map[string]int
	total    int
	maxPerIP int
	maxTotal int
}

func newConnLimiter(maxPerIP, maxTotal int) *connLimiter {
	return &connLimiter{
		perIP:    make(map[string]int),
		maxPerIP: maxPerIP,
		maxTotal: maxTotal,
	}
}

// acquire reserves a connection slot for ip, or returns false when either the
// per-IP or global cap is reached. Every successful acquire must be paired with
// a release.
func (c *connLimiter) acquire(ip string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.total >= c.maxTotal || c.perIP[ip] >= c.maxPerIP {
		return false
	}
	c.perIP[ip]++
	c.total++
	return true
}

func (c *connLimiter) release(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.perIP[ip] > 0 {
		c.perIP[ip]--
		c.total--
		if c.perIP[ip] == 0 {
			delete(c.perIP, ip)
		}
	}
}
