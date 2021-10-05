package errors

import (
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/messages"
)

//goland:noinspection SpellCheckingInspection

// NewResourceNotFoundError returns a new ErrNotFound error with kind
// KindResourceNotFound and the given message.
func NewResourceNotFoundError(message string, details Details) error {
	return Error{
		Code:    ErrNotFound,
		Kind:    KindResourceNotFound,
		Message: message,
		Details: details,
	}
}

// NewForbiddenMessageError creates a new ErrProtocolViolation error with kind
// KindForbiddenMessage.
func NewForbiddenMessageError(messageType messages.MessageType, content json.RawMessage) error {
	return Error{
		Code:    ErrProtocolViolation,
		Kind:    KindForbiddenMessage,
		Message: fmt.Sprintf("forbidden message type: %s", messageType),
		Details: Details{
			"messageType": messageType,
			"content":     content,
		},
	}
}
