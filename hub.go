package main

import "github.com/aljo242/shmeeload.xyz/internal/log"

// roomMessage is a message bound for a single room.
type roomMessage struct {
	room string
	body []byte
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

		case client := <-h.unregister:
			if clients, ok := h.rooms[client.room]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.room)
					}
				}
			}

		case m := <-h.broadcast:
			if h.store != nil {
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

// stop signals run to close all client connections and return.
func (h *Hub) stop() { close(h.quit) }
