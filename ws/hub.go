package ws

import (
	"context"
	"github.com/LeFinal/masc-server/client"
	"github.com/LeFinal/masc-server/logging"
)

// Hub holds all active clients and manages centralized receiving and sending.
type Hub struct {
	// clientListener is used for notifying of new clients or unregistered ones.
	clientListener client.Listener
	// clients holds all online clients.
	clients map[*Client]struct{}
	// register receives when a Client wants to register itself.
	register chan *Client
	// unregister receives when a Client wants to unregister itself.
	unregister chan *Client
}

// NewHub creates a new Hub. Start it with Hub.Run.
func NewHub(clientListener client.Listener) *Hub {
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
		case c := <-h.register:
			// Register client.
			h.clients[c] = struct{}{}
			logging.WSLogger.WithField("client_id", c.ID).Info("client connected")
			go h.clientListener.AcceptClient(ctx, c.Client)
		case c := <-h.unregister:
			// Unregister client.
			if _, ok := h.clients[c]; ok {
				h.clientListener.SayGoodbyeToClient(ctx, c.Client)
				delete(h.clients, c)
				logging.WSLogger.WithField("client_id", c.ID).Info("client disconnected")
				// Close the send-channel which leads to stopping the write-pump.
				close(c.Send)
			}
		}
	}
}
