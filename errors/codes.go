package errors

type Code string

const (
	ErrBadRequest        Code = "bad-request"
	ErrCommunication     Code = "communication"
	ErrProtocolViolation Code = "protocol-violation"
	ErrFatal             Code = "fatal"
	ErrNotFound          Code = "not-found"
	ErrInternal          Code = "internal"
	ErrUnexpected        Code = "unexpected"
)

type Kind string

const (
	// KindActorAlreadyHired is used when someone wants to hire an acting.Actor that
	// has already been hired.
	KindActorAlreadyHired Kind = "actor-already-hired"
	// KindActorNotHired is used when someone instructs an acting.Actor to send a
	// message although not hired yet.
	KindActorNotHired      Kind = "actor-not-hired"
	KindCommunicationWrite Kind = "communication-write"
	KindEncodeJSON         Kind = "encode-json"
	// KindMalformedID is used when a passed ID is not in uuid.UUID format.
	KindMalformedID Kind = "malformed-id"
	KindMissingID   Kind = "missing-id"
	// KindForbiddenMessage is used when the protocol is being violated due to a
	// message with currently forbidden type.
	KindForbiddenMessage Kind = "protocol-violation"
	KindDecodeJSON       Kind = "parse-request-body-as-json"
	KindResourceNotFound Kind = "resource-not-found"
	KindUUIDGenFail      Kind = "uuid-gen-fail"
	KindUnexpected       Kind = "unexpected"
	// KindUnknownActor is used when a message with an unknown actor is received.
	KindUnknownActor Kind = "unknown-actor"
	// KindUnknownDevice is used when an unknown device is being requested.
	KindUnknownDevice Kind = "unknown-device"
	// KindUnknownRole is used when a role is unknown.
	KindUnknownRole Kind = "unknown-role"
)
