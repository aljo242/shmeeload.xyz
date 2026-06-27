package main

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/caddyserver/certmagic"
)

// acmeTLSConfig returns a tls.Config that obtains and auto-renews publicly
// trusted certificates from Let's Encrypt for the configured domains, storing
// them on disk (a persistent volume) so they survive restarts. The TLS-ALPN-01
// challenge is answered on the same :443 listener, so no extra port is needed
// (we never forward :80).
func acmeTLSConfig(cfg Config) (*tls.Config, error) {
	if len(cfg.Domains) == 0 {
		return nil, fmt.Errorf("acme enabled but no domains configured")
	}
	if cfg.ACMEDir != "" {
		certmagic.Default.Storage = &certmagic.FileStorage{Path: cfg.ACMEDir}
	}
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = cfg.ACMEEmail
	// Only the TLS-ALPN-01 challenge on :443; :80 is not forwarded.
	certmagic.DefaultACME.DisableHTTPChallenge = true
	if cfg.ACMEStaging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	} else {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	}

	magic := certmagic.NewDefault()
	if err := magic.ManageSync(context.Background(), cfg.Domains); err != nil {
		return nil, fmt.Errorf("managing certificates: %w", err)
	}

	tlsCfg := magic.TLSConfig()
	// Serve HTTP/2 and HTTP/1.1 while keeping certmagic's acme-tls/1 entry (last)
	// so the ALPN challenge still resolves.
	tlsCfg.NextProtos = append([]string{"h2", "http/1.1"}, tlsCfg.NextProtos...)
	return tlsCfg, nil
}
