package main

import (
	"fmt"
	"log"
)

// DebugLogln is a simple utility for conditional logging of bonus info
func DebugLogln(toggle bool, msg string) {
	if toggle {
		log.Println(msg)
	}
}

// DebugPrintln is a simple utility for conditional logging of bonus info
func DebugPrintln(toggle bool, msg string) {
	if toggle {
		fmt.Println(msg)
	}
}
