package main

import (
	"encoding/json"
	"sort"

	"github.com/aljo242/shmeeload.xyz/internal/log"
)

// rosterPrefix marks a frame as a presence (room roster) update rather than a
// chat message. It is a NUL byte, which never appears in a chat line.
const rosterPrefix = "\x00"

// roomMessage is a message bound for a single room. System messages (joins,
// leaves) are broadcast but not persisted, so they stay out of replayed history.
type roomMessage struct {
	room   string
	body   []byte
	system bool
}

// Hub keeps the set of connected clients per room and fans messages out within
// a room. Messages are persisted (when a store is configured) before broadcast.
type Hub struct {
	// room name -> set of clients in that room
	rooms map[string]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan roomMessage
	quit       chan struct{}

	store *chatStore // nil disables persistence
}

func newHub(store *chatStore) *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan roomMessage),
		quit:       make(chan struct{}),
		store:      store,
	}
}

func (h *Hub) run() {
	for {
		select {
		case <-h.quit:
			// On shutdown close every client's send channel so its writePump
			// emits a close frame and the connection is torn down.
			for _, clients := range h.rooms {
				for client := range clients {
					close(client.send)
				}
			}
			return

		case client := <-h.register:
			clients := h.rooms[client.room]
			if clients == nil {
				clients = make(map[*Client]bool)
				h.rooms[client.room] = clients
			}
			clients[client] = true
			h.broadcastRoster(client.room)

		case client := <-h.unregister:
			if clients, ok := h.rooms[client.room]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.room)
					} else {
						h.broadcastRoster(client.room)
					}
				}
			}

		case m := <-h.broadcast:
			if h.store != nil && !m.system {
				if err := h.store.save(m.room, m.body); err != nil {
					log.Error("error persisting chat message", "room", m.room, "err", err)
				}
			}
			clients := h.rooms[m.room]
			for client := range clients {
				select {
				case client.send <- m.body:
				default:
					close(client.send)
					delete(clients, client)
				}
			}
			if len(clients) == 0 {
				delete(h.rooms, m.room)
			}
		}
	}
}

// broadcastRoster sends the room's current set of (unique, sorted) member names
// to everyone in it, as a roster frame. Not persisted: it is live presence.
func (h *Hub) broadcastRoster(room string) {
	clients := h.rooms[room]
	seen := make(map[string]bool, len(clients))
	names := make([]string, 0, len(clients))
	for client := range clients {
		if !seen[client.name] {
			seen[client.name] = true
			names = append(names, client.name)
		}
	}
	sort.Strings(names)
	payload, err := json.Marshal(names)
	if err != nil {
		log.Error("error encoding roster", "room", room, "err", err)
		return
	}
	frame := append([]byte(rosterPrefix), payload...)
	for client := range clients {
		select {
		case client.send <- frame:
		default:
			close(client.send)
			delete(clients, client)
		}
	}
}

// stop signals run to close all client connections and return.
func (h *Hub) stop() { close(h.quit) }
