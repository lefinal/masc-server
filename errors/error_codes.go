package errors

const (
	// Events
	EventNotFoundError ErrorCode = "event.not-found"
	// Gatekeeping
	IdAlreadyTakenError ErrorCode = "gatekeeping.id-already-taken"
	// Networking
	UnknownMessageTypeError ErrorCode = "networking.unknown-message-type"
)
