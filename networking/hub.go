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

	// The new closed clients.
	ClosedClients chan *Client
}

// NewHub creates a new Hub using the given http server listen address.
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

// Run runs the Hub. This should be run in a go routine as the method will manage
// all Client stuff related to registering, unregistering and handling message
// broadcasts.
func (h *Hub) Run() {
	go runHubServer(h)
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
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

func (h *Hub) unregisterClient(c *Client) {
	if _, ok := h.clients[c]; !ok {
		logging.Error("client requested unregister although not registered")
		return
	}
	delete(h.clients, c)
	close(c.send)
	logging.Info("Connection to client closed.")
	h.ClosedClients <- c
}
