// Provide basic message functionality.

package messages

import (
	"encoding/json"
	"github.com/LeFinal/masc-server/errors"
	"github.com/google/uuid"
)

// MessageType is the type of message and serves for using the correct parsing
// method.
type MessageType string

// DeviceID is used to identify a Device.
type DeviceID uuid.UUID

func (id DeviceID) String() string {
	return uuid.UUID(id).String()
}

// ActorID is used to identify an acting.Actor.
type ActorID uuid.UUID

func (id ActorID) String() string {
	return uuid.UUID(id).String()
}

// MessageContainer is a container for all messages that are sent and received.
// It holds some meta information as well as the actual payload.
type MessageContainer struct {
	// MessageType is the type of the message.
	MessageType MessageType `json:"message_type"`
	// DeviceID is the id that is used for identifying the devices.Device the
	// message belongs to.
	DeviceID DeviceID `json:"device_id"`
	// ActorID is the optional ID of the actor. This is used for concurrent
	// communication with actors that use the same device.
	ActorID ActorID `json:"actor_id,omitempty"`
	// Content is the actual message content.
	Content json.RawMessage `json:"content"`
}

// All message types.
const (
	// MessageTypeOK is used only for confirmation of actions that do not require a
	// detailed response.
	MessageTypeOK MessageType = "ok"
	// MessageTypeError is used for error messages. The content is being set to the
	// detailed error.
	MessageTypeError MessageType = "error"
	// MessageTypeHello is received with MessageHello for saying hello to the
	// server.
	MessageTypeHello MessageType = "hello"
	// MessageTypeWelcome is sent to the client when he is welcomed at the server.
	// Used with MessageWelcome.
	MessageTypeWelcome MessageType = "welcome"
	// MessageTypeGoAway is sent to the client when he wants to say hello with an
	// unknown device ID.
	MessageTypeGoAway MessageType = "go-away"
	// MessageTypeGetDevices is received when devices are requested.
	MessageTypeGetDevices MessageType = "get-devices"
	// MessageTypeDeviceList is used with MessageDeviceList as an answer to
	// MessageTypeGetDevices.
	MessageTypeDeviceList MessageType = "device-list"
	// MessageTypeWelcomeDevice is used with MessageWelcomeDevice for accepting new
	// devices and allowing them to communicate with MASC.
	MessageTypeWelcomeDevice MessageType = "welcome-device"
)

// MessageError is used with MessageTypeError for errors that need to be sent to devices.
type MessageError struct {
	// Code is the error code from errors.Error.
	Code string `json:"code"`
	// Kind is the kind code from errors.Error.
	Kind string `json:"kind"`
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
	return MessageError{
		Code:    string(e.Code),
		Kind:    string(e.Kind),
		Err:     e.Error(),
		Message: e.Message,
		Details: e.Details,
	}
}
