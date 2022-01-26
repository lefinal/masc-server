package errors

type Code string

const (
	ErrAborted           Code = "aborted"
	ErrBadRequest        Code = "bad-request"
	ErrCommunication     Code = "communication"
	ErrProtocolViolation Code = "protocol-violation"
	ErrFatal             Code = "fatal"
	ErrNotFound          Code = "not-found"
	ErrInternal          Code = "internal"
	ErrSadLife           Code = "sad-life"
	ErrUnexpected        Code = "unexpected"
)
