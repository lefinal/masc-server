package ws

import (
	"context"
	"github.com/LeFinal/masc-server/client"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

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
		c := &Client{
			Client: &client.Client{
				ID:      uuid.New().String(),
				Send:    make(chan []byte, 256),
				Receive: make(chan []byte, 256),
			},
			hub:        hub,
			connection: conn,
		}
		// Use the client's hub so that the reference from the handler can be dropped.
		c.hub.register <- c
		// Power the pumps.
		go c.writePump()
		go c.readPump(ctx)
	}
}
