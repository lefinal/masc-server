package messages

import (
	"github.com/LeFinal/masc-server/pkg/errors"
)

// The message types
const (
	MsgTypeError MessageType = "error"
)

// ErrorMessage is being sent if a client needs to be informed of an error.
type ErrorMessage struct {
	MessageMeta `json:"meta"`
	ErrorCode   errors.ErrorCode `json:"error_code"`
	Message     string           `json:"message"`
}
