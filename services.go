package main

import (
	"context"
	"net"
	"sync"
	"time"
)

// serviceCache holds health for the services the site can check itself (as
// opposed to the ones the game host pushes). Right now that is Pi-hole, which
// runs alongside the site, so the site probes it directly.
type serviceCache struct {
	piholeDNS string

	mu       sync.RWMutex
	piholeUp bool
	checked  bool
}

func newServiceCache(piholeDNS string) *serviceCache {
	c := &serviceCache{piholeDNS: piholeDNS}
	if piholeDNS != "" {
		go c.loop()
	}
	return c
}

func (c *serviceCache) loop() {
	c.refresh()
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		c.refresh()
	}
}

func (c *serviceCache) refresh() {
	up := checkPiholeDNS(c.piholeDNS, 2*time.Second)
	c.mu.Lock()
	c.piholeUp = up
	c.checked = true
	c.mu.Unlock()
}

// services returns the site-checked service statuses, or nil before the first
// check completes.
func (c *serviceCache) services() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.checked {
		return nil
	}
	status := "down"
	if c.piholeUp {
		status = "up"
	}
	return map[string]string{"pihole": status}
}

// checkPiholeDNS reports whether the Pi-hole resolver at addr answers a query
// for "pi.hole" (its own always-present record). This tests the function that
// matters (DNS resolution) and is independent of the Pi-hole admin API version.
func checkPiholeDNS(addr string, timeout time.Duration) bool {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, network, addr)
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := r.LookupHost(ctx, "pi.hole")
	return err == nil
}
