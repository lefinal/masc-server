package networking

// The client is based on https://github.com/gorilla/websocket/blob/master/examples/chat/client.go.

import (
	"bytes"
	"github.com/LeFinal/masc-server/util"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// SendMessage pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type ClientEventType string

const (
	ClientEventTypeIncomingMessage = "incoming-message"
	ClientEventTypeClosed          = "closed"
)

type ClientEvent struct {
	EventType ClientEventType
	Message   string
}

type OutboundMessage struct {
	Message []byte
}

type InboundMessage struct {
	Message string
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type MessagePort interface {
	util.Identifiable
	sendMessage(msg OutboundMessage)
	run()
	close()
}

type messageReceiver interface {
	receiveMessage(msg InboundMessage)
}

type ClientSocketPort struct {
	id  uuid.UUID
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	receiver messageReceiver

	receivedMessageCount int
	sentMessageCount     int
}

type Client interface {
	SendMessage(msg OutboundMessage)
	Inbox() chan InboundMessage
}

// NetClient is a middleman between the websocket connection and the hub.
type NetClient struct {
	net MessagePort

	// Buffered channel for inbound messages.
	Receive chan InboundMessage

	ReceivedMessageCount int
	SentMessageCount     int
}

// NewClient creates a new client with given message port.
func NewClient(port MessagePort) *NetClient {
	return &NetClient{
		net:                  port,
		Receive:              make(chan InboundMessage, 256),
		ReceivedMessageCount: 0,
		SentMessageCount:     0,
	}
}

func (sp *ClientSocketPort) sendMessage(msg OutboundMessage) {
	sp.send <- msg.Message
}

func (sp *ClientSocketPort) run() {
	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go sp.writePump()
	go sp.readPump()
}

func (sp *ClientSocketPort) Identify() uuid.UUID {
	return sp.id
}

func (sp *ClientSocketPort) close() {
	close(sp.send)
}

// SendMessage sends a message to the network port.
func (c *NetClient) SendMessage(msg OutboundMessage) {
	c.net.sendMessage(msg)
}

func (c *NetClient) Inbox() chan InboundMessage {
	return c.Receive
}

func (c *NetClient) receiveMessage(msg InboundMessage) {
	c.Receive <- msg
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (sp *ClientSocketPort) readPump() {
	defer func() {
		sp.hub.unregister <- sp
		if err := sp.conn.Close(); err != nil {
			clientLogger.Error("could not close client")
		}
	}()
	sp.conn.SetReadLimit(maxMessageSize)
	if err := sp.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		clientLogger.Errorf("could not sead read deadline for client: ", err)
	}
	sp.conn.SetPongHandler(func(string) error {
		if err := sp.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			clientLogger.Errorf("could not set read deadline for client: %s", err)
		}
		return nil
	})
	for {
		_, message, err := sp.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		sp.receivedMessageCount++
		sp.receiver.receiveMessage(InboundMessage{
			Message: string(message),
		})
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (sp *ClientSocketPort) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := sp.conn.Close(); err != nil {
			clientLogger.Errorf("could not close client: ", err)
		}
	}()
	for {
		select {
		case message, ok := <-sp.send:
			if err := sp.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				clientLogger.Errorf("could not set write deadline for client: %s", err)
			}
			if !ok {
				// The hub closed the channel.
				if err := sp.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					clientLogger.Errorf("could not write close message to client: %s", err)
				}
				return
			}

			w, err := sp.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, errWrite := w.Write(message); errWrite != nil {
				clientLogger.Errorf("could not write message to client: %s", err)
			}

			sp.sentMessageCount++

			// Add queued chat messages to the current websocket message.
			n := len(sp.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write(newline); err != nil {
					clientLogger.Errorf("could not write new line for queued chat messages to client: %s", err)
				}
				if _, err := w.Write(<-sp.send); err != nil {
					clientLogger.Errorf("could not write queued chat message to client: %s", err)
				}
				sp.sentMessageCount++
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := sp.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				clientLogger.Errorf("could not set write deadline for client: %s", err)
			}
			if err := sp.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		clientLogger.Errorf("upgrade connection for client: %s", err)
		return
	}
	socketPort := &ClientSocketPort{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	socketPort.hub.register <- NewClient(socketPort)
	go socketPort.run()
}
