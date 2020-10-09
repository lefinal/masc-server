// Provide basic message functionality.

package messages

import (
	"encoding/json"
	"github.com/google/uuid"
)

type MessageType string

// Message is the basic interface that provides basic functionality for messages.
// MessageType retrieves the type of the message in lowercase.
// MessageId retrieves the id.
type Message interface {
	MessageType()
	MessageId()
}

const MsgTypeOk MessageType = "ok"

// MessageMeta provides basic information that is used in each message.
type MessageMeta struct {
	Type     string    `json:"type"`      // The message type
	DeviceId uuid.UUID `json:"device_id"` // The device id which is used for all it's performers
}

func (meta MessageMeta) MessageType() MessageType {
	return MessageType(meta.Type)
}

// GeneralMessage is mainly used for checking meta information upon receiving.
type GeneralMessage struct {
	MessageMeta `json:"meta"`
	Payload     json.RawMessage `json:"payload"`
}

// MessageContainer is a container for outbound messages that also holds the message type. This makes it faster
// for the DeviceNetworkPort to build the meta data as the type is already known.
type MessageContainer struct {
	MessageType MessageType
	Payload     interface{}
}

// NewMessageContainerForError creates a new message container with message type MsgTypeError and the given
// payload.
func NewMessageContainerForError(payload interface{}) MessageContainer {
	return MessageContainer{
		MessageType: MsgTypeError,
		Payload:     payload,
	}
}

type OkMessage struct {
	Message string `json:"message"`
}

func (t MessageType) String() string {
	return string(t)
}
