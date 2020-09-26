package model

// The message types
const (
	MsgTypeError MessageType = "error"
)

type ErrorCode string

// ErrorMessage is being sent if a client needs to be informed of an error.
type ErrorMessage struct {
	MessageMeta `json:"meta"`
	ErrorCode   ErrorCode `json:"error_code"`
	Message     string    `json:"message"`
}
