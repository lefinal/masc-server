// Provide basic message functionality.

package messages

import "github.com/google/uuid"

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

func (meta MessageMeta) MessageType() string {
	return meta.Type
}

// GeneralMessage is mainly used for checking meta information upon receiving.
type GeneralMessage struct {
	MessageMeta `json:"meta"`
	Payload     interface{} `json:"payload"`
}

type OkMessage struct {
	Message string `json:"message"`
}
