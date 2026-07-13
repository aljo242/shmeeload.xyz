package main

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"
)

// certChecker tracks how many days until the site's TLS certificate expires, by
// dialing the domain and reading the served leaf cert. Caddy terminates TLS, so
// this reports the cert clients actually get. daysLeft is -1 until the first
// successful check; ok is false when the last check could not read a cert.
type certChecker struct {
	domain string

	mu       sync.RWMutex
	daysLeft int
	ok       bool
}

func newCertChecker(domain string) *certChecker {
	c := &certChecker{domain: domain, daysLeft: -1}
	if domain != "" {
		go c.loop()
	}
	return c
}

func (c *certChecker) loop() {
	c.refresh()
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for range t.C {
		c.refresh()
	}
}

func (c *certChecker) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	d := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 5 * time.Second},
		Config:    &tls.Config{ServerName: c.domain, MinVersion: tls.VersionTLS12},
	}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(c.domain, "443"))
	if err != nil {
		c.set(-1, false)
		return
	}
	defer conn.Close()
	tc, ok := conn.(*tls.Conn)
	if !ok {
		c.set(-1, false)
		return
	}
	certs := tc.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		c.set(-1, false)
		return
	}
	c.set(int(time.Until(certs[0].NotAfter).Hours()/24), true)
}

func (c *certChecker) set(days int, ok bool) {
	c.mu.Lock()
	c.daysLeft, c.ok = days, ok
	c.mu.Unlock()
}

func (c *certChecker) days() (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.daysLeft, c.ok
}
