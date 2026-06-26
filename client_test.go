package main

import (
	"net/http/httptest"
	"testing"
)

func TestSameOriginCheck(t *testing.T) {
	cases := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{"no origin header is allowed", "shmee.lan", "", true},
		{"matching origin", "shmee.lan", "https://shmee.lan", true},
		{"matching origin, case-insensitive", "shmee.lan", "https://SHMEE.LAN", true},
		{"cross-origin is rejected", "shmee.lan", "https://evil.example", false},
		{"malformed origin is rejected", "shmee.lan", "%zz", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/chat/ws", nil)
			r.Host = c.host
			if c.origin != "" {
				r.Header.Set("Origin", c.origin)
			}
			if got := sameOriginCheck(r); got != c.want {
				t.Errorf("sameOriginCheck(host=%q, origin=%q) = %v, want %v", c.host, c.origin, got, c.want)
			}
		})
	}
}
