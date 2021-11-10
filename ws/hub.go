package ws

import (
	"context"
	"github.com/LeFinal/masc-server/logging"
)

// Hub holds all active clients and manages centralized receiving and sending.
type Hub struct {
	// clientListener is used for notifying of new clients or unregistered ones.
	clientListener ClientListener
	// clients holds all online clients.
	clients map[*Client]struct{}
	// register receives when a Client wants to register itself.
	register chan *Client
	// unregister receives when a Client wants to unregister itself.
	unregister chan *Client
}

// NewHub creates a new Hub. Start it with Hub.Run.
func NewHub(clientListener ClientListener) *Hub {
	return &Hub{
		clientListener: clientListener,
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		clients:        make(map[*Client]struct{}),
	}
}

// Run starts the Hub. It blocks so you need to start a goroutine.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			// Register client.
			h.clients[client] = struct{}{}
			logging.WSLogger.Infof("client %v connected", client.ID)
			go h.clientListener.AcceptClient(ctx, client)
		case client := <-h.unregister:
			// Unregister client.
			if _, ok := h.clients[client]; ok {
				h.clientListener.SayGoodbyeToClient(client)
				delete(h.clients, client)
				logging.WSLogger.Infof("client %v disconnected", client.ID)
				// Close the send-channel which leads to stopping the write-pump.
				close(client.Send)
			}
		}
	}
}
