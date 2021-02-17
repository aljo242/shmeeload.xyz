package main

import (
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aljo242/web_serve/romanNumerals"

	"github.com/glendc/go-external-ip"
	//"github.com/gorilla/mux"
)

const (
	DefaultPort = "80"

	DefaultHost = "localhost"
)

var (
	// Port of the HTTP Server
	Port = "80"

	// Host name of the HTTP Server
	Host = "192.168.1.18"
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

func getExternalIP() (string, error) {
	// create the default consensus
	// using the default configuratoin and no logger
	consensus := externalip.DefaultConsensus(nil, nil)
	// Get my IP
	// which is never <nil> when err is <nil>
	extIP, err := consensus.ExternalIP()
	if err != nil {
		return "", err
	}

	return extIP.String(), nil
}

func getHostInfo() map[int]string {
	m := make(map[int]string)

	name, err := os.Hostname()
	if err != nil {
		log.Fatal("Error getting Hostname : ", err)
		return m
	}
	fmt.Printf("Hostname: %v\n", name)

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Error getting net interfaces : ", err)
		return m
	}

	extIP, err := getExternalIP()
	if err != nil {
		log.Fatal("Error getting external IP address: ", err)
		return m
	}
	fmt.Printf("External IP address: %v\n", extIP)

	m[0] = DefaultHost
	index := 1

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Fatal("Error getting interface addresses: ", err)
			return m
		}

		if !strings.Contains(i.Name, "lo") {
			for _, addr := range addrs {
				var ip net.IP // IP address
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}

				m[index] = ip.String()
				index++
			}
		}
	}

	return m
}

func selectHost(ipMap map[int]string) string {
	fmt.Printf("Choose a host to use:\n")
	for ind, ip := range ipMap {
		fmt.Printf("%d\t%v\n", ind, ip)
	}

	var userInd int = 0
	_, err := fmt.Scanln(&userInd)
	if err != nil {
		log.Fatal("Error scanning user input: ", err)
		return ""
	}

	ret, ok := ipMap[userInd]
	if !ok {
		log.Fatal("Error getting ip from ip map: ", err)
		return ""
	}

	return ret
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

func runRomanServer() {
	// http package has methods for dealing with requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			romanGet(w, r)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 - Bad Request"))
		}
	})

	s := &http.Server{
		Addr:           ":8000",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}

func runDemoServer() {
	m := getHostInfo()
	Host := selectHost(m)
	fmt.Printf("Host: %v\n", Host)

	web := webServer{
		name:        "Demo Web Server",
		author:      "Cozart Shmoopler",
		connections: 1,
	}

	addr := Host + ":" + Port
	fmt.Printf("Serving to %v...\n", addr)

	err := http.ListenAndServe(addr, web)
	if err != nil {
		log.Fatal("Error Starting the HTTP Server : ", err)
		return
	}
}

func main() {
	runRomanServer()
}
