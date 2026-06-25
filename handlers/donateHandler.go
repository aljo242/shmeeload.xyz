package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// cryptoAddr maps a lowercase currency code to its public donation address.
var cryptoAddr = map[string]string{
	"btc": "3P9F4yxfpmwrLnSzg4c1SCFr177yBUx3WF",
	"ltc": "MPS7pQ3eKyYqdeXDS79ezJ8T5dT31Hxzap",
	"eth": "0xd78804789051626fa7dbf0f9bfba21a5c697c8b8",
	"ada": "addr1q93g6h6cj6kdvjad9tuykr76f77r2pth4r6c9l2sq7amzypkxvjyelkkwprsxm0cpcqgagc9fzgpxqnmgk7ejkfm389sa56wmh",
}

// DonateHandler serves the public donation address for the requested currency.
// Unknown currencies return 404. The address comes from a fixed internal map, so
// it is safe to embed directly in the response.
func DonateHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currency := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "DonateHandler").Str("Currency", currency).Msg("incoming request")

		addr, ok := cryptoAddr[currency]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", cacheControl(cacheMaxAge))
		fmt.Fprintf(w, "<h1>%s</h1>\n", addr)
	}
}
