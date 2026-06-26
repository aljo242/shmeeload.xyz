package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"
)

func TestIPRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	l := newIPRateLimiter(1, 3) // 1/s sustained, burst of 3
	for i := 0; i < 3; i++ {
		if !l.allow("1.2.3.4") {
			t.Fatalf("request %d within burst should be allowed", i+1)
		}
	}
	if l.allow("1.2.3.4") {
		t.Error("request beyond the burst should be blocked")
	}
	// A different IP has its own bucket.
	if !l.allow("5.6.7.8") {
		t.Error("a different IP should not be throttled")
	}
}

func TestIPRateLimiterMiddleware(t *testing.T) {
	l := newIPRateLimiter(1, 1)
	h := l.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "9.9.9.9:1111"
	h.ServeHTTP(first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("first request = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, req)
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("second request = %d, want 429", second.Code)
	}
}

func TestConnLimiterCaps(t *testing.T) {
	c := newConnLimiter(2, 3) // 2 per IP, 3 total

	for i := 0; i < 2; i++ {
		if !c.acquire("a") {
			t.Fatalf("connection %d for an IP should be allowed", i+1)
		}
	}
	if c.acquire("a") {
		t.Error("third connection for the same IP should be rejected (per-IP cap)")
	}

	if !c.acquire("b") {
		t.Fatal("a second IP should get a slot (under the global cap)")
	}
	if c.acquire("b") {
		t.Error("connection beyond the global cap should be rejected")
	}

	// Releasing frees a slot for that IP and the total.
	c.release("a")
	if !c.acquire("a") {
		t.Error("after release the IP should get a slot again")
	}
}

// TestIPRateLimiterRefill uses a synctest fake-time bubble to verify the token
// bucket refills over time, without any real sleeping.
func TestIPRateLimiterRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newIPRateLimiter(1, 2) // 1 token/sec, burst of 2
		defer l.Stop()

		for i := 0; i < 2; i++ {
			if !l.allow("ip") {
				t.Fatalf("request %d within the burst should be allowed", i+1)
			}
		}
		if l.allow("ip") {
			t.Fatal("a third immediate request should be blocked")
		}

		time.Sleep(time.Second) // fake clock advances; one token refills
		if !l.allow("ip") {
			t.Error("a request should be allowed after the bucket refills")
		}
	})
}

// TestSweepLoopEvictsIdle drives the real background sweeper with fake time: an
// idle entry should be evicted once the TTL ticker fires.
func TestSweepLoopEvictsIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newIPRateLimiter(1, 1)
		defer l.Stop()

		l.allow("ip")
		time.Sleep(2 * ipEntryTTL) // let the sweep ticker fire past the TTL
		synctest.Wait()            // let the sweeper finish its tick

		l.mu.Lock()
		_, present := l.entries["ip"]
		l.mu.Unlock()
		if present {
			t.Error("an idle entry should have been swept")
		}
	})
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.7:54321"
	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP = %q, want 203.0.113.7", got)
	}
}
