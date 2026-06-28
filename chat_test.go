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
	hub := newHub(nil)
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
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/chat/ws?room=general"
}

func newChatServerWithStore(t *testing.T, store *chatStore) *httptest.Server {
	t.Helper()
	hub := newHub(store)
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

func TestWebSocketUnknownRoom(t *testing.T) {
	srv := newChatTestServer(t)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/chat/ws?room=does-not-exist"
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("dialing an unknown room should be rejected")
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown-room status = %v, want 404", resp)
	}
}

func TestWebSocketHistoryReplay(t *testing.T) {
	store, err := newChatStore(t.TempDir() + "/chat.db")
	if err != nil {
		t.Fatalf("newChatStore: %v", err)
	}
	defer store.close()
	srv := newChatServerWithStore(t, store)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/chat/ws?room=general"

	// First client posts a message, which should be persisted.
	c1, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial c1: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // registered
	if err := c1.WriteMessage(websocket.TextMessage, []byte("alex: hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	time.Sleep(200 * time.Millisecond) // let it persist
	_ = c1.Close()

	// A new client joining the room should be replayed that message as history.
	c2, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial c2: %v", err)
	}
	defer func() { _ = c2.Close() }()
	if err := c2.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	_, msg, err := c2.ReadMessage()
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	if string(msg) != "alex: hello" {
		t.Errorf("history = %q, want \"alex: hello\"", msg)
	}
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
