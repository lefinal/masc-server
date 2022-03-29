package ws

import (
	"context"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lefinal/masc-server/client"
	"go.uber.org/zap"
	"log"
	"net/http"
)

// HandleWS handles websocket requests. The passed context is used in order to
// stop all remaining read-pumps.
func HandleWS(ctx context.Context, hub *Hub) http.HandlerFunc {
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
		clientID := uuid.New().String()
		c := &Client{
			Client: &client.Client{
				ID:      clientID,
				Send:    make(chan []byte, 256),
				Receive: make(chan []byte, 256),
			},
			hub:        hub,
			connection: conn,
			logger:     hub.logger.With(zap.String("client_id", clientID)),
		}
		// Use the client's hub so that the reference from the handler can be dropped.
		c.hub.register <- c
		// Power the pumps.
		go c.writePump()
		go c.readPump(ctx)
	}
}
