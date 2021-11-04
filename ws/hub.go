package ws

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
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// Register client.
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			// Unregister client.
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// Close the send-channel which leads to stopping the write-pump.
				close(client.Send)
			}
		}
	}
}
