package errors

// Code is an error code in order to identify the type of Error.
type Code string

const (
	// ErrAborted is used when an action is aborted. Mostly used with aborted
	// context.Context.
	ErrAborted Code = "aborted"
	// ErrBadRequest is used when an invalid request or input was given.
	ErrBadRequest Code = "bad-request"
	// ErrCommunication is used for failing communication.
	ErrCommunication Code = "communication"
	// ErrProtocolViolation is used when the communication protocol is violated.
	ErrProtocolViolation Code = "protocol-violation"
	// ErrFatal is used for fatal errors.
	ErrFatal Code = "fatal"
	// ErrNotFound is used when a requested resource was not found.
	ErrNotFound Code = "not-found"
	// ErrInternal is used for most internal errors.
	ErrInternal Code = "internal"
	// ErrUnexpected is used for errors that have not been assigned another Code.
	ErrUnexpected Code = "unexpected"
)
