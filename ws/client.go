package ws

import (
	"bytes"
	"context"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
	// id is a temporary id assigned to the Client.
	id uuid.UUID
	// hub is the actual websocket hub which is used for registering and
	// unregistering.
	hub *Hub
	// connection is the actual websocket connection.
	connection *websocket.Conn
	// Send is the channel for outgoing messages are passed to.
	Send chan []byte
	// Receive is the channel for incoming messages.
	Receive chan []byte
}

// logger returns a logrus.Entry with the Client id as field.
func (c *Client) logger() *logrus.Entry {
	return logging.WSLogger.WithField("client", c.id)
}

// readPump forwards messages from the websocket connection to the hub.
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		err := c.connection.Close()
		if err != nil {
			c.logger().Warn(errors.Wrap(err, "close connection"))
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
				c.logger().Warn(errors.Wrap(err, "unexpected close"))
			}
			break
		}
		// Trim.
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// Forward.
		select {
		case <-ctx.Done():
			c.logger().WithField("message", message).Warn("dropping message due to ctx done")
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
			c.logger().Warnf(errors.Wrap(err, "close connection").Error())
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
					c.logger().Error(errors.Wrap(err, "write close message"))
				}
				return
			}
			// Write message.
			nextWriter, err := c.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				// We expect the read pump to fail as well.
				c.logger().Warn(errors.Wrap(err, "create writer for text message"))
				return
			}
			c.logger().Debugf("write message")
			_, err = nextWriter.Write(message)
			if err != nil {
				c.logger().Warnf(errors.Wrap(err, "write text message").Error())
			}
			// Add queued messages to the current websocket message.
			remainingMessages := len(c.Send)
			for i := 0; i < remainingMessages; i++ {
				_, err = nextWriter.Write(newline)
				if err != nil {
					c.logger().Warnf(errors.Wrap(err, "write new line").Error())
				}
				_, err = nextWriter.Write(<-c.Send)
				if err != nil {
					c.logger().Warnf(errors.Wrap(err, "write queued message").Error())
				}
			}
			// Close writer.
			if err := nextWriter.Close(); err != nil {
				c.logger().Warnf(errors.Wrap(err, "close next writer").Error())
				return
			}
		case <-pingTicker.C:
			// Send ping.
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger().Warnf(errors.Wrap(err, "write ping").Error())
				return
			}
		}
	}
}
