package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDonateHandler(t *testing.T) {
	t.Run("known currency returns its address", func(t *testing.T) {
		for currency, addr := range cryptoAddr {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/donate/"+currency, nil)
			DonateHandler(0)(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("%s: status = %d, want 200", currency, rr.Code)
			}
			if !strings.Contains(rr.Body.String(), addr) {
				t.Errorf("%s: body %q does not contain address %q", currency, rr.Body.String(), addr)
			}
		}
	})

	t.Run("unknown currency returns 404", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/donate/doge", nil)
		DonateHandler(0)(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
	})
}
