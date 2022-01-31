package ws

import (
	"bytes"
	"context"
	"github.com/LeFinal/masc-server/client"
	"github.com/LeFinal/masc-server/logging"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"time"
)

const (
	// writeTimeout is the timeout for writing a message to the peer.
	writeTimeout = 10 * time.Second
	// pingInterval is the interval in which pings are sent to the peer. Must be
	// less than pongTimeout.
	pingInterval = (pongTimeout * 9) / 10
	// pongTimeout is the timeout for waiting for the next pong message from the
	// peer. Must be greater than pingInterval.
	pongTimeout = 60 * time.Second
	// maxMessageSize is the maximum message size allowed from peer.
	maxMessageSize = 16384
)

var (
	// newLine is used for separating messages in writer.
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a holds the websocket connection and is being used by Hub.
type Client struct {
	*client.Client
	// hub is the actual websocket hub which is used for registering and
	// unregistering.
	hub *Hub
	// connection is the actual websocket connection.
	connection *websocket.Conn
	logger     *zap.Logger
}

// readPump forwards messages from the websocket connection to the hub.
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		err := c.connection.Close()
		if err != nil {
			c.logger.Debug("close connection failed", zap.Error(err))
		}
	}()
	c.connection.SetReadLimit(maxMessageSize)
	_ = c.connection.SetReadDeadline(time.Now().Add(pongTimeout))
	// Handle received pong.
	c.connection.SetPongHandler(func(string) error {
		_ = c.connection.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})
	for {
		// Read next message.
		_, message, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Debug("unexpected close", zap.Error(err))
			}
			break
		}
		// Trim.
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// Forward.
		select {
		case <-ctx.Done():
			c.logger.Debug("dropping message due to ctx done",
				zap.Any("message", message))
		case c.Receive <- message:
		}
	}
}

// writePump forwards outgoing messages from the hub to the websocket
// connection. We do not pass a context.Context here because the hub will close
// the Send-channel which will lead to termination, anyways.
func (c *Client) writePump() {
	pingTicker := time.NewTicker(pingInterval)
	defer func() {
		// Stop ping ticker in order to avoid ticker leak.
		pingTicker.Stop()
		// Close connection.
		err := c.connection.Close()
		if err != nil {
			c.logger.Debug("close connection failed", zap.Error(err))
		}
	}()
	for {
		select {
		case message, ok := <-c.Send:
			// Set write timeout.
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeTimeout))
			// Check if connection close is requested from hub.
			if !ok {
				err := c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					logging.WSLogger.Debug("write close message failed", zap.Error(err))
					return
				}
				return
			}
			// Write message.
			nextWriter, err := c.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				// We expect the read pump to fail as well.
				c.logger.Warn("create writer for text message failed", zap.Error(err))
				return
			}
			_, err = nextWriter.Write(message)
			if err != nil {
				c.logger.Debug("write text message failed", zap.Error(err))
			}
			// Close writer.
			if err := nextWriter.Close(); err != nil {
				c.logger.Warn("close next writer failed", zap.Error(err))
				return
			}
		case <-pingTicker.C:
			// Send ping.
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Debug("write ping message failed", zap.Error(err))
				return
			}
		}
	}
}
