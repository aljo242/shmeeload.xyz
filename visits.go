package main

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// visitCounter is a small persistent hit counter backed by a text file.
// Reads the last value on startup, increments in memory, and best-effort
// persists via a temp-file + rename on each increment.
type visitCounter struct {
	mu   sync.Mutex
	n    int64
	path string
}

func newVisitCounter(path string) *visitCounter {
	c := &visitCounter{path: path}
	if b, err := os.ReadFile(path); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil {
			c.n = v
		}
	}
	return c
}

// Inc increments the counter, persists it, and returns the new value.
func (c *visitCounter) Inc() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	tmp := c.path + ".tmp"
	if os.WriteFile(tmp, []byte(strconv.FormatInt(c.n, 10)), 0o600) == nil {
		_ = os.Rename(tmp, c.path)
	}
	return c.n
}

// Get returns the current value without incrementing.
func (c *visitCounter) Get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
