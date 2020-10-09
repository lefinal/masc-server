package messages

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
)

// The message types
const (
	MsgTypeError MessageType = "error"
)

// ErrorMessage is being sent if a client needs to be informed of an error.
type ErrorMessage struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// NewErrorMessageFromMascError creates a new error messages that is built from a given errors.MascError.
func NewErrorMessageFromMascError(err *errors.MascError) ErrorMessage {
	message := err.Message
	if message == "" {
		message = fmt.Sprintf("An error occurred: %s", err.ErrorCode.String())
	}
	return ErrorMessage{
		ErrorCode: err.ErrorCode.String(),
		Message:   message,
	}
}
