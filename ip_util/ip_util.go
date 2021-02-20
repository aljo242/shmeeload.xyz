package ip_util

import (
	"net"
	"os"
	"strings"

	externalip "github.com/glendc/go-external-ip"
)

const (
	defaultHost string = "localhost"
)

// ExternalIP returns the current :exernal IP address and an error
func ExternalIP() (string, error) {
	// create the default consensus
	// using the default configuration and no logger
	consenus := externalip.DefaultConsensus(nil, nil)
	// Get my IP
	// which is never <nil> when err is <nil>
	extIP, err := consenus.ExternalIP()
	if err != nil {
		return "", err
	}

	return extIP.String(), nil
}

type Host struct {
	Name        string
	InternalIPs map[int]string
	ExternalIP  string
}

func HostInfo() (Host, error) {
	h := Host{}
	m := make(map[int]string)

	name, err := os.Hostname()
	if err != nil {
		return Host{}, err
	}
	h.Name = name

	extIP, err := ExternalIP()
	if err != nil {
		return Host{}, err
	}
	h.ExternalIP = extIP

	ifaces, err := net.Interfaces()
	if err != nil {
		return Host{}, err
	}

	m[0] = defaultHost
	index := 1
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return Host{}, err
		}

		// exclude internal device
		if !strings.Contains(i.Name, "lo") {
			for _, addr := range addrs {

			}
		}
	}

	return h, nil
}
