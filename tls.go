package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// ensureCert makes sure a TLS certificate/key pair exists at certFile/keyFile,
// generating a long-lived self-signed certificate for the given hosts if either
// is missing. The certificate is self-signed (its own root), so each client
// trusts it once; persisting the files (e.g. on a volume) keeps it stable across
// restarts so the trust holds.
func ensureCert(certFile, keyFile string, hosts []string) error {
	if certExists(certFile) && certExists(keyFile) {
		return nil
	}
	if len(hosts) == 0 {
		return fmt.Errorf("no TLS hosts configured")
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generating serial: %w", err)
	}

	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: hosts[0], Organization: []string{"shmeeload"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshaling key: %w", err)
	}

	if err := writePEM(certFile, "CERTIFICATE", der, 0o644); err != nil {
		return err
	}
	return writePEM(keyFile, "EC PRIVATE KEY", keyDER, 0o600)
}

func certExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
