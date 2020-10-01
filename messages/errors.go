package messages

// The message types
const (
	MsgTypeError MessageType = "error"
)

// ErrorMessage is being sent if a client needs to be informed of an error.
type ErrorMessage struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}
