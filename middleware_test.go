package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	called := false
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("inner handler was not called")
	}
	for k, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := rr.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing Content-Security-Policy header")
	}
}

func TestApexDomain(t *testing.T) {
	if got := apexDomain([]string{"www.djinntek.space", "djinntek.space"}); got != "djinntek.space" {
		t.Errorf("apexDomain = %q, want djinntek.space", got)
	}
	if got := apexDomain(nil); got != "" {
		t.Errorf("apexDomain(nil) = %q, want empty", got)
	}
}

func TestRedirectToApex(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("www redirects to apex, preserving path and query", func(t *testing.T) {
		h := redirectToApex("djinntek.space")(ok)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://www.djinntek.space/resume/home?x=1", nil)
		req.Host = "www.djinntek.space"
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusMovedPermanently {
			t.Fatalf("status = %d, want 301", rr.Code)
		}
		if loc := rr.Header().Get("Location"); loc != "https://djinntek.space/resume/home?x=1" {
			t.Errorf("Location = %q", loc)
		}
	})

	t.Run("apex passes through", func(t *testing.T) {
		h := redirectToApex("djinntek.space")(ok)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://djinntek.space/", nil)
		req.Host = "djinntek.space"
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("empty apex never redirects", func(t *testing.T) {
		h := redirectToApex("")(ok)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://www.example.com/", nil)
		req.Host = "www.example.com"
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (no public domain configured)", rr.Code)
		}
	})
}
