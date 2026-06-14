package ws

import (
	"log"
	"sync"

	"salesmee/internal/chatpb"

	"google.golang.org/protobuf/proto"
)

type Hub struct {
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			for _, room := range client.rooms {
				if h.rooms[room] == nil {
					h.rooms[room] = make(map[*Client]bool)
				}
				h.rooms[room][client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			for _, room := range client.rooms {
				if h.rooms[room] != nil {
					delete(h.rooms[room], client)
					if len(h.rooms[room]) == 0 {
						delete(h.rooms, room)
					}
				}
			}
			close(client.send)
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Broadcast(room string, frame *chatpb.WsFrame, exclude *Client) {
	data, err := proto.Marshal(frame)
	if err != nil {
		log.Printf("ws marshal error: %v", err)
		return
	}
	h.mu.RLock()
	clients := h.rooms[room]
	for client := range clients {
		if client == exclude {
			continue
		}
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.rooms, room)
			}
		}
	}
	h.mu.RUnlock()
}
