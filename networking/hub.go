package networking

import (
	"github.com/LeFinal/masc-server/config"
	"github.com/LeFinal/masc-server/util"
)

// The hub is based on https://github.com/gorilla/websocket/blob/master/examples/chat/hub.go.

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Server address
	addr string

	// Whether to run a websocket server.
	socketMode bool

	// Registered clients.
	clients map[*NetClient]bool

	// Register requests from the clients.
	register chan *NetClient

	// Unregister requests from clients.
	unregister chan util.Identifiable

	// New registered clients.
	NewClients chan *NetClient

	// The new closed clients.
	ClosedClients chan *NetClient
}

// NewHub creates a new Hub using the given http server listen address.
func NewHub(config config.NetworkConfig) *Hub {
	return &Hub{
		addr:       config.Address,
		register:   make(chan *NetClient),
		unregister: make(chan util.Identifiable),
		clients:    make(map[*NetClient]bool),
		NewClients: make(chan *NetClient, 16),
		socketMode: config.Address != "",
	}
}

// Run runs the Hub. This should be run in a go routine as the method will manage
// all NetClient stuff related to registering, unregistering and handling message
// broadcasts.
func (h *Hub) Run() {
	if h.socketMode {
		go runHubServer(h)
	}
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case identifiable := <-h.unregister:
			h.unregisterClient(identifiable)
		}
	}
}

func (h *Hub) registerClient(c *NetClient) {
	hubLogger.Info("New incoming client connection.")
	h.clients[c] = true
	h.NewClients <- c
}

func (h *Hub) unregisterClient(i util.Identifiable) {
	// Find the client by id.
	var client *NetClient
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
