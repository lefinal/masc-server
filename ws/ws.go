package ws

import (
	"context"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

// ClientListener provides methods for accepting new clients and unregister
// events.
type ClientListener interface {
	// AcceptClient is called when a new Client connects.
	AcceptClient(ctx context.Context, client *Client)
	// SayGoodbyeToClient is called when a Client's connection has been closed.
	SayGoodbyeToClient(client *Client)
}

// HandleWS handles websocket requests. The passed context is used in order to
// stop all remaining read-pumps.
func HandleWS(hub *Hub, ctx context.Context) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		client := &Client{
			ID:         uuid.New(),
			hub:        hub,
			connection: conn,
			Send:       make(chan []byte, 256),
			Receive:    make(chan []byte, 256),
		}
		// Use the client's hub so that the reference from the handler can be dropped.
		client.hub.register <- client
		// Power the pumps.
		go client.writePump()
		go client.readPump(ctx)
	}
}
