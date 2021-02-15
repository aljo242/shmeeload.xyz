package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	
	"github.com/glendc/go-external-ip"
)

const (
	// Port of the HTTP Server
	Port = "80"
)

var (
	
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
	consensus := externalip.DefaultConsensus(nil,nil)
	// Get my IP
	// which is never <nil> when err is <nil>
	extIP, err := consensus.ExternalIP()
	if err != nil {
		return "", err
	}

	return extIP.String(), nil
}

func getHostInfo() {
	name, err := os.Hostname()
	if err != nil {
		log.Fatal("Error getting Hostname : ", err)
		return
	}
	fmt.Printf("Hostname: %v\n", name)

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Error getting net interfaces : ", err)
		return
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Fatal("Error getting interface addresses: ", err)
			return
		}

		for _, addr := range addrs {
			var ip net.IP // IP address
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// TODO more with the IP address
			fmt.Printf("Interface: %v\n", i)
			fmt.Printf("IP address: %v\n", ip)
		}
	}

	extIP, err := getExternalIP()
	if err != nil {
		log.Fatal("Error getting external IP address: ", err)
		return
	} 

	fmt.Printf("External IP address: %v\n", extIP)
	// Host = extIP
}

func main() {
	getHostInfo()

	web := webServer{
		name:        "Demo Web Server",
		author:      "Cozart Shmoopler",
		connections: 1,
	}

	addr := Host + ":" + Port
	fmt.Printf("Serving to %v\n", addr)

	err := http.ListenAndServe(addr, web)
	if err != nil {
		log.Fatal("Error Starting the HTTP Server : ", err)
		return
	}
}
