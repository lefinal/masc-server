package client

import (
	"context"
)

// Client is a holds the connection and is used by gatekeeping.Gatekeeper as
// well as ws.Hub.
type Client struct {
	// ID is a temporary id assigned to the Client.
	ID string
	// Send is the channel for outgoing messages are passed to.
	Send chan []byte
	// Receive is the channel for incoming messages.
	Receive chan []byte
}

// Listener provides methods for accepting new clients and unregister events.
type Listener interface {
	// AcceptClient is called when a new Client connects.
	AcceptClient(ctx context.Context, client *Client)
	// SayGoodbyeToClient is called when a Client's connection has been closed.
	SayGoodbyeToClient(ctx context.Context, client *Client)
}
