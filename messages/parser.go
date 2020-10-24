package messages

import (
	"encoding/json"
	"github.com/LeFinal/masc-server/errors"
)

// ParseMessage parses a given message and returns the meta and payload.
func ParseMessage(msg []byte) (MessageMeta, interface{}, *errors.MascError) {
	generalMessage, parseGeneralErr := parseGeneralMessage(msg)
	if parseGeneralErr != nil {
		return MessageMeta{}, nil,
			errors.NewMascErrorFromError("parse general message", errors.ParseMetaErrorError, parseGeneralErr)
	}
	// Parse the payload.
	payload, messageTypeErr := CreateMessageContainerForType(generalMessage.MessageMeta.MessageType())
	if messageTypeErr != nil {
		return MessageMeta{}, nil, errors.PropagateMascError("get container type for message", messageTypeErr)
	}
	if err := json.Unmarshal(generalMessage.Payload, payload); err != nil {
		return MessageMeta{}, nil,
			errors.NewMascErrorFromError("parse message payload", errors.ParsePayloadErrorError, err)
	}
	// Parsing went fine.
	return generalMessage.MessageMeta, payload, nil
}

func parseGeneralMessage(msg []byte) (GeneralMessage, error) {
	var generalMessage GeneralMessage
	if err := json.Unmarshal(msg, &generalMessage); err != nil {
		return GeneralMessage{}, err
	}
	return generalMessage, nil
}

func convertToRawMessage(i interface{}) json.RawMessage {
	msg, _ := json.Marshal(i)
	return msg
}

// MarshalMessage does simple message marshalling.
func MarshalMessage(msg GeneralMessage) ([]byte, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// MarshalMessageMust does simple message marshalling and panics if an error occurs.
func MarshalMessageMust(msg GeneralMessage) []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

// MarshalPayload converts the passed payload to a JSON raw message.
func MarshalPayload(payload interface{}) (json.RawMessage, *errors.MascError) {
	marshalledPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.NewMascErrorFromError("marshal", errors.MarshalPayloadErrorError, err)
	}
	return marshalledPayload, nil
}

// MarshalMessageFromMetaAndPayloadMust marshals a message with given meta and payload to string.
func MarshalMessageFromMetaAndPayloadMust(meta MessageMeta, payload interface{}) string {
	return string(MarshalMessageMust(GeneralMessage{
		MessageMeta: meta,
		Payload:     convertToRawMessage(payload),
	}))
}
