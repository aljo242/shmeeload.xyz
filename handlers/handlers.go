package handlers

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aljo242/web_serve/romanNumerals"
)

func romanGet(w http.ResponseWriter, r *http.Request) {
	urlPathElements := strings.Split(r.URL.Path, "/")

	log.Println(r.URL.Path)
	log.Println(urlPathElements)

	if urlPathElements[1] == "roman_number" {
		number, err := strconv.Atoi(strings.TrimSpace(urlPathElements))
		if err != nil {
			log.Fatal("Error getting integer from URL string : ", err)
			return
		}

		if number == 0 || number > 10 {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 - Not Found"))
		} else {
			fmt.Fprintf(w, "%v", html.EscapeString(romanNumerals.Numerals[number]))
		}

	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Bad Request"))
	}
}
