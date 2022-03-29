// Provide basic message functionality.

package event

import (
	"github.com/eclipse/paho.golang/paho"
	"github.com/lefinal/masc-server/errors"
)

// Standalone events.
const ()

// Event is a container for a parsed paho.Publish payload.
type Event[T any] struct {
	// Publish is the raw publish event.
	Publish *paho.Publish
	// Payload is the parsed payload of Publish when used with appropriate
	// subscribers that perform unmarshalling.
	Payload T
}

// EmptyEvent is needed when no content is expected.
type EmptyEvent struct{}

// ErrorEventPayload is used with MessageTypeError for errors that need to be sent to devices.
type ErrorEventPayload struct {
	// Code is the error code from errors.Error.
	Code string `json:"code"`
	// Err is the error from errors.Error.
	Err string `json:"err"`
	// Message is the message from errors.Error.
	Message string `json:"message"`
	// Details are error details from errors.Error.
	Details map[string]interface{} `json:"details"`
}

// ErrorEventPayloadFromError creates a ErrorEventPayload from the given error.
func ErrorEventPayloadFromError(err error) ErrorEventPayload {
	e, _ := errors.Cast(err)
	if !errors.BlameUser(err) {
		return ErrorEventPayload{
			Code:    string(e.Code),
			Message: "internal server error",
		}
	}
	return ErrorEventPayload{
		Code:    string(e.Code),
		Err:     e.Error(),
		Message: e.Message,
		Details: e.Details,
	}
}
