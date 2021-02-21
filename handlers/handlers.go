package handlers

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aljo242/web_serve/ip_util"
	"github.com/aljo242/web_serve/romanNumerals"
	"github.com/gorilla/mux"
)

var (
	port string = "80"
)

type webServer struct {
	name        string
	author      string
	connections int
}

func (this webServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "This is a Simple HTTP Web Server!")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	//w.Header().Set("Name", this.name)
	//w.Header().Set("Author", this.author)
}

func romanGet(w http.ResponseWriter, r *http.Request) {
	urlPathElements := strings.Split(r.URL.Path, "/")

	log.Println(r.URL.Path)
	log.Println(urlPathElements)

	if urlPathElements[1] == "roman_number" {
		number, err := strconv.Atoi(strings.TrimSpace(urlPathElements[2]))
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

// RunRomanServer runs our roman numeral dummy server
func RunRomanServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			romanGet(w, r) // pass onto Get sub-handler
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 - Bad Request"))
		}
	})

	s := &http.Server{
		Addr:           ":8000",
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}

// RunDemoServer runs a very basic server with IP utils
func RunDemoServer() {
	h, err := ip_util.HostInfo()
	if err != nil {
		log.Fatal("Error creating host struct : ", err)
		return
	}

	hostIP, err := ip_util.SelectHost(h.InternalIPs)
	if err != nil {
		log.Fatal("Error chosing host IP : ", err)
		return
	}

	addr := hostIP + ":" + port
	web := webServer{
		name:        "Demo Web Server",
		author:      "Cozart Shmoopler",
		connections: 1,
	}

	log.Printf("main: serving to %v...\n", addr)

	err = http.ListenAndServe(addr, web)
	if err != nil {
		log.Fatal("Error starting the HTTP server : ", err)
		return
	}
}

// ArticleHandler handles our Gorilla Server Handler
func ArticleHandler(w http.ResponseWriter, r *http.Request) {
	// mux.Vars returns all path parameters as a map
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK) // TODO do not accept all request
	fmt.Fprintf(w, "Category is: %v\n", vars["category"])
	fmt.Fprintf(w, "ID is %v\n", vars["id"])
}

// HomeHandler serves the home.html file
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// this page currently only serves html resources
	if r.Method == "GET" {
		wantFile := "./html/home.html"
		if _, err := os.Stat(wantFile); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			log.Fatalf("Error finding file %v : %v", wantFile, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		http.ServeFile(w, r, wantFile)

	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// RedirectHome redirects urls to the address to be served by HomeHandler
func RedirectHome(w http.ResponseWriter, r *http.Request) {
	// redirect to home
	http.Redirect(w, r, "http://shmeeload.xyz/home", http.StatusFound)
}
