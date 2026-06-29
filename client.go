package main

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aljo242/shmeeload.xyz/internal/log"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

const (
	// chatTimeFormat prefixes every chat line with a server-side timestamp.
	chatTimeFormat = "Jan 2 15:04"

	// maxNameLen caps a client-supplied display name.
	maxNameLen = 24

	// Time allowed to write a message to peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pong wait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte("\n")
	space   = []byte(" ")
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     sameOriginCheck,
}

// chatLine prefixes text with a server timestamp, so message times are
// consistent and survive in persisted history.
func chatLine(text string) []byte {
	return []byte("[" + time.Now().Format(chatTimeFormat) + "] " + text)
}

// sanitizeName makes a client-supplied name safe to display: printable,
// single-line, length-capped, defaulting to "anon".
func sanitizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f { // strip control chars (including newlines)
			return -1
		}
		return r
	}, name)
	// Cap length before trimming, so a truncation can't leave trailing space.
	if r := []rune(name); len(r) > maxNameLen {
		name = string(r[:maxNameLen])
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "anon"
	}
	return name
}

// sameOriginCheck only allows websocket upgrades whose Origin host matches the
// request host, preventing cross-site websocket hijacking. Requests without an
// Origin header (non-browser clients) are allowed.
func sameOriginCheck(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub *Hub

	// websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// room this client is joined to; name is its display name (from the join URL).
	room string
	name string

	// ip and conns release this connection's slot in the per-IP/global cap when
	// the connection closes.
	ip    string
	conns *connLimiter

	// msgLimiter throttles inbound chat messages from this connection.
	msgLimiter *rate.Limiter
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine.
// The application ensures that there is at most one reader on a connection
// by executing all reads from this goroutine
func (c *Client) readPump() {
	defer func() {
		// Announce the departure to the room, then unregister. Both are guarded
		// so they don't block once the hub has stopped (server shutdown).
		select {
		case c.hub.broadcast <- roomMessage{room: c.room, body: chatLine("* " + c.name + " left *"), system: true}:
		case <-c.hub.quit:
		}
		select {
		case c.hub.unregister <- c:
		case <-c.hub.quit:
		}
		c.conns.release(c.ip)
		if err := c.conn.Close(); err != nil {
			log.Error("error closing WebSocket connection", "err", err)
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Error("error setting WebSocket read deadline", "err", err)
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("unexpected websocket close", "err", err)
			}
			break
		}

		// Drop messages from a connection that exceeds its rate instead of
		// disconnecting, so a brief burst is tolerated but a flood cannot spam
		// the broadcast.
		if !c.msgLimiter.Allow() {
			continue
		}

		message = bytes.TrimSpace(bytes.ReplaceAll(message, newline, space))
		// The server stamps the time and the (trusted) connection name, so they
		// cannot be spoofed per message and are consistent across clients.
		c.hub.broadcast <- roomMessage{room: c.room, body: chatLine(c.name + ": " + string(message))}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			log.Error("error closing WebSocket connection", "err", err)
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Error("error setting WebSocket write deadline", "err", err)
				return
			}

			if !ok {
				// the hub closed the channel
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Error("error writing CloseMessage on WebSocket", "err", err)
				}
				return
			}

			// One frame per message, so the client can distinguish roster
			// frames from chat lines (they are never concatenated).
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Error("error writing WebSocket message", "err", err)
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Error("error setting WebSocket write deadline", "err", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error("error writing PingMessage to WebSocket", "err", err)
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer. The room comes from the
// "room" query parameter and must be one of the curated rooms.
func serveWs(hub *Hub, conns *connLimiter, rooms map[string]bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		room := r.URL.Query().Get("room")
		if !rooms[room] {
			http.Error(w, "unknown room", http.StatusNotFound)
			return
		}
		name := sanitizeName(r.URL.Query().Get("name"))

		ip := clientIP(r)
		// Cap concurrent connections before upgrading so a flood is rejected cheaply.
		if !conns.acquire(ip) {
			http.Error(w, "too many connections", http.StatusTooManyRequests)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			conns.release(ip)
			log.Error("error upgrading to websocket", "err", err)
			return
		}

		// Replay recent history before the pumps start, so each past message is
		// its own frame (the writePump batches whatever is queued together).
		if hub.store != nil {
			history, err := hub.store.recent(room, chatHistoryLimit)
			if err != nil {
				log.Error("error loading chat history", "room", room, "err", err)
			} else {
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				for _, body := range history {
					if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
						break
					}
				}
			}
		}

		client := &Client{
			hub:        hub,
			conn:       conn,
			send:       make(chan []byte, 256),
			room:       room,
			name:       name,
			ip:         ip,
			conns:      conns,
			msgLimiter: rate.NewLimiter(wsMsgPerSec, wsMsgBurst),
		}

		// Start the pumps, then register: the writePump must be ready to drain the
		// send channel before the hub can target this client with a broadcast.
		go client.writePump()
		go client.readPump()
		client.hub.register <- client

		// Announce the arrival to the room (broadcast, not persisted).
		select {
		case hub.broadcast <- roomMessage{room: room, body: chatLine("* " + name + " joined *"), system: true}:
		case <-hub.quit:
		}
	}
}
