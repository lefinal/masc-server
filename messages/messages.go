// Provide basic message functionality.

package messages

import (
	"encoding/json"
	"github.com/LeFinal/masc-server/errors"
)

// MessageType is the type of message and serves for using the correct parsing
// method.
type MessageType string

// DeviceID is a UUID that is used to identify a Device.
type DeviceID string

// ActorID is a UUID that is used to identify an acting.Actor.
type ActorID string

// UserID is a UUID that is used to identify a stores.Player.
type UserID string

// PlayerRank is the rank of a player.
type PlayerRank int

// Role is used for provided functionality.
type Role string

// ProviderID is the id of an entity that is being set by the provider and
// remembered. Through using the provider id, the provider always knows what
// entity is being addressed.
type ProviderID string

// MessageContainer is a container for all messages that are sent and received.
// It holds some meta information as well as the actual payload.
type MessageContainer struct {
	// MessageType is the type of the message.
	MessageType MessageType `json:"message_type"`
	// DeviceID is the id that is used for identifying the devices.Device the
	// message belongs to.
	DeviceID DeviceID `json:"device_id,omitempty"`
	// ActorID is the optional ID of the actor. This is used for concurrent
	// communication with actors that use the same device.
	ActorID ActorID `json:"actor_id,omitempty"`
	// Content is the actual message content.
	Content json.RawMessage `json:"content,omitempty"`
}

// All message types.
const (
	// MessageTypeError is used for error messages. The content is being set to the
	// detailed error.
	MessageTypeError MessageType = "error"
	// MessageTypeHello is received with MessageHello for saying hello to the
	// server.
	MessageTypeHello MessageType = "hello"
	// MessageTypeOK is used only for confirmation of actions that do not require a
	// detailed response.
	MessageTypeOK MessageType = "ok"
	// MessageTypeWelcome is sent to the client when he is welcomed at the server.
	// Used with MessageWelcome.
	MessageTypeWelcome MessageType = "welcome"
)

// MessageError is used with MessageTypeError for errors that need to be sent to devices.
type MessageError struct {
	// Code is the error code from errors.Error.
	Code string `json:"code"`
	// Err is the error from errors.Error.
	Err string `json:"err"`
	// Message is the message from errors.Error.
	Message string `json:"message"`
	// Details are error details from errors.Error.
	Details map[string]interface{} `json:"details"`
}

// MessageErrorFromError creates a MessageError from the given error.
func MessageErrorFromError(err error) MessageError {
	e, _ := errors.Cast(err)
	if !errors.BlameUser(err) {
		return MessageError{
			Code:    string(e.Code),
			Message: "internal server error",
		}
	}
	return MessageError{
		Code:    string(e.Code),
		Err:     e.Error(),
		Message: e.Message,
		Details: e.Details,
	}
}
