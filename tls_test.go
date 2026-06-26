package main

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureCert(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	if err := ensureCert(certFile, keyFile, []string{"shmee.lan", "127.0.0.1"}); err != nil {
		t.Fatalf("ensureCert: %v", err)
	}

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadX509KeyPair: %v", err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if len(leaf.DNSNames) == 0 || leaf.DNSNames[0] != "shmee.lan" {
		t.Errorf("DNSNames = %v, want shmee.lan", leaf.DNSNames)
	}
	if len(leaf.IPAddresses) == 0 {
		t.Error("expected the 127.0.0.1 SAN")
	}
	if leaf.NotAfter.Before(time.Now().AddDate(9, 0, 0)) {
		t.Errorf("NotAfter = %v, want ~10y out", leaf.NotAfter)
	}

	t.Run("idempotent: existing files are not regenerated", func(t *testing.T) {
		before, _ := os.ReadFile(certFile)
		if err := ensureCert(certFile, keyFile, []string{"shmee.lan"}); err != nil {
			t.Fatalf("second ensureCert: %v", err)
		}
		after, _ := os.ReadFile(certFile)
		if string(before) != string(after) {
			t.Error("ensureCert regenerated an existing certificate")
		}
	})

	t.Run("no hosts is an error", func(t *testing.T) {
		if err := ensureCert(filepath.Join(dir, "c2.pem"), filepath.Join(dir, "k2.pem"), nil); err == nil {
			t.Error("expected an error when no hosts are configured")
		}
	})
}
