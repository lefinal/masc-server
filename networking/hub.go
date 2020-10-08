package networking

import "github.com/LeFinal/masc-server/logging"

// The hub is based on https://github.com/gorilla/websocket/blob/master/examples/chat/hub.go.

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Server address
	addr string

	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// New registered clients.
	NewClients chan *Client
}

func NewHub(addr string) *Hub {
	return &Hub{
		addr:       addr,
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		NewClients: make(chan *Client),
	}
}

func (h *Hub) Run() {
	go runHubServer(h)
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; !ok {
				logging.Error("client requested unregister although not registered")
				continue
			}
			delete(h.clients, client)
			close(client.send)
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) registerClient(c *Client) {
	logging.Info("New incoming client connection.")
	h.clients[c] = true
	h.NewClients <- c
}
