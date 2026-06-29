package main

import (
	"testing"
	"time"
)

// TestHubBroadcast verifies a registered client receives broadcasts and that
// unregistering closes its send channel.
func TestHubBroadcast(t *testing.T) {
	h := newHub(nil)
	go h.run()

	c := &Client{hub: h, send: make(chan []byte, 8)}
	h.register <- c
	<-c.send // drain the roster (presence) frame sent on join

	h.broadcast <- roomMessage{body: []byte("hello")}
	select {
	case msg := <-c.send:
		if string(msg) != "hello" {
			t.Fatalf("received %q, want %q", msg, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("registered client did not receive the broadcast")
	}

	h.unregister <- c
	select {
	case _, ok := <-c.send:
		if ok {
			t.Fatal("send channel should be closed after unregister")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("send channel was not closed after unregister")
	}
}

// TestHubDropsSlowClient verifies the hub drops (and closes) a client whose send
// buffer is full rather than blocking the broadcast loop.
func TestHubDropsSlowClient(t *testing.T) {
	h := newHub(nil)
	go h.run()

	c := &Client{hub: h, send: make(chan []byte, 1)}
	h.register <- c
	<-c.send // drain the roster frame, leaving the size-1 buffer empty

	// Re-fill the buffer so the hub's non-blocking send cannot enqueue the
	// broadcast and must take the default case: close and drop the client.
	c.send <- []byte("backlog")
	h.broadcast <- roomMessage{body: []byte("dropped")}

	// Registering another client forces the single hub goroutine through another
	// select iteration, which guarantees the broadcast above is fully processed
	// (and c already dropped) before we inspect c.send.
	h.register <- &Client{hub: h, send: make(chan []byte, 8)}

	if got, ok := <-c.send; !ok || string(got) != "backlog" {
		t.Fatalf("first read = (%q, %v), want (\"backlog\", true)", got, ok)
	}
	select {
	case _, ok := <-c.send:
		if ok {
			t.Fatal("expected slow client's send channel to be closed after drop")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hub did not close the dropped client's channel")
	}
}
