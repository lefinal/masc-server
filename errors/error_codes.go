package errors

const (
	// Events
	EventNotFoundError ErrorCode = "event.not-found"
	// Gatekeeping
	IdAlreadyTakenError ErrorCode = "gatekeeping.id-already-taken"
	// Messages
	ParseMetaErrorError              ErrorCode = "messages.parse-meta-error"
	ParsePayloadErrorError           ErrorCode = "messages.parse-payload-error"
	InvalidPayloadMessageFormatError ErrorCode = "messages.invalid-payload-message-format"
	// Networking
	UnknownMessageTypeError ErrorCode = "networking.unknown-message-type"
)
