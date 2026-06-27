package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gorilla/websocket"
)

// newChatTestServer spins up the real router (with the abuse-control middleware
// and the chat hub) over httptest's loopback listener.
func newChatTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	hub := newHub()
	go hub.run()
	site, err := newStaticSite(fstest.MapFS{}, 60)
	if err != nil {
		t.Fatalf("newStaticSite: %v", err)
	}
	srv := httptest.NewServer(buildRouter(Config{Port: "0"}, hub, site))
	t.Cleanup(func() {
		srv.Close()
		hub.stop()
	})
	return srv
}

func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/chat/ws"
}

func TestWebSocketConnectionCap(t *testing.T) {
	srv := newChatTestServer(t)
	url := wsURL(srv)

	// Hold the maximum number of connections open for one IP (the loopback).
	conns := make([]*websocket.Conn, 0, wsMaxPerIP)
	for i := 0; i < wsMaxPerIP; i++ {
		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial %d should succeed: %v", i+1, err)
		}
		conns = append(conns, c)
	}
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()

	// The next connection from the same IP must be rejected before upgrade.
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("the over-cap connection should have been rejected")
	}
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("over-cap status = %v, want 429", resp)
	}
}

func TestWebSocketEcho(t *testing.T) {
	srv := newChatTestServer(t)
	c, _, err := websocket.DefaultDialer.Dial(wsURL(srv), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	// Let the hub finish registering this client before broadcasting.
	time.Sleep(100 * time.Millisecond)

	if err := c.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := c.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	_, msg, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(msg) != "hello" {
		t.Errorf("echoed %q, want \"hello\"", msg)
	}
}

func TestWebSocketMessageRateLimit(t *testing.T) {
	srv := newChatTestServer(t)
	c, _, err := websocket.DefaultDialer.Dial(wsURL(srv), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	time.Sleep(100 * time.Millisecond) // registered

	// Flood far past the per-connection rate; most should be dropped.
	const sent = 40
	for i := 0; i < sent; i++ {
		if err := c.WriteMessage(websocket.TextMessage, []byte("m")); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	if err := c.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	got := 0
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break // read deadline reached: we have drained what was broadcast
		}
		got += bytes.Count(msg, []byte("m")) // writePump batches messages with newlines
	}
	if got == 0 {
		t.Error("expected at least the burst to get through")
	}
	if got > wsMsgBurst+5 {
		t.Errorf("received %d messages from a flood of %d; rate limit did not drop enough", got, sent)
	}
}
