package errors

const (
	// Events
	EventNotFoundError ErrorCode = "event.not-found"
	// Gatekeeping
	IdAlreadyTakenError ErrorCode = "gatekeeping.id-already-taken"
	// MessageTypeNotAllowedError is used when a message has a message type that is currently not allowed.
	MessageTypeNotAllowedError ErrorCode = "gatekeeping.message-type-not-allowed"
	// InvalidDeviceIdError is used when a message is received containing a wrong device id.
	InvalidDeviceIdError ErrorCode = "gatekeeping.invalid-device-id"
	// Messages
	ParseMetaErrorError    ErrorCode = "messages.parse-meta-error"
	ParsePayloadErrorError ErrorCode = "messages.parse-payload-error"
	// MarshalPayloadErrorError is used when the payload cannot be marshalled.
	MarshalPayloadErrorError ErrorCode = "messages.marshal-payload-error"
	// MarshalMessageErrorError is used when the message cannot be marshalled.
	MarshalMessageErrorError         ErrorCode = "messages.marshal-message"
	InvalidPayloadMessageFormatError ErrorCode = "messages.invalid-payload-message-format"
	// Networking
	UnknownMessageTypeError ErrorCode = "networking.unknown-message-type"
)
