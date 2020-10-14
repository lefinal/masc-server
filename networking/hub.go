package networking

import (
	"github.com/LeFinal/masc-server/util"
)

// The hub is based on https://github.com/gorilla/websocket/blob/master/examples/chat/hub.go.

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Server address
	addr string

	// Registered clients.
	clients map[*Client]bool

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan util.Identifiable

	// New registered clients.
	NewClients chan *Client

	// The new closed clients.
	ClosedClients chan *Client
}

// NewHub creates a new Hub using the given http server listen address.
func NewHub(addr string) *Hub {
	return &Hub{
		addr:       addr,
		register:   make(chan *Client),
		unregister: make(chan util.Identifiable),
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
		case identifiable := <-h.unregister:
			h.unregisterClient(identifiable)
		}
	}
}

func (h *Hub) registerClient(c *Client) {
	hubLogger.Info("New incoming client connection.")
	h.clients[c] = true
	h.NewClients <- c
}

func (h *Hub) unregisterClient(i util.Identifiable) {
	// Find the client by id.
	var client *Client
	for c, _ := range h.clients {
		if c.net.Identify() == i.Identify() {
			client = c
			break
		}
	}
	if client == nil {
		hubLogger.Error("client requested unregister although not registered")
		return
	}
	delete(h.clients, client)
	client.net.close()
	hubLogger.Info("Connection to client closed.")
	h.ClosedClients <- client
}
